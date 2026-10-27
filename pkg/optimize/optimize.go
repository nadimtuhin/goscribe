package optimize

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nadimtuhin/goscribe/pkg/runner"
)

var presetHeights = map[string]int{
	"360p":  360,
	"480p":  480,
	"720p":  720,
	"1080p": 1080,
	"1440p": 1440,
	"2k":    1440,
	"4k":    2160,
	"2160p": 2160,
}

var (
	heightRegex = regexp.MustCompile(`^(\d+)p?$`)
	dimRegex    = regexp.MustCompile(`^(\d+)[x:](\d+)$`)
	queueMu     sync.Mutex
)

func ParseResolution(res string) (width, height int, err error) {
	norm := strings.ToLower(strings.TrimSpace(res))
	if h, ok := presetHeights[norm]; ok {
		return 0, h, nil
	}
	if m := heightRegex.FindStringSubmatch(norm); len(m) == 2 {
		h, _ := strconv.Atoi(m[1])
		if h > 0 {
			return 0, h, nil
		}
	}
	if m := dimRegex.FindStringSubmatch(norm); len(m) == 3 {
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		if w > 0 && h > 0 {
			return w, h, nil
		}
	}
	return 0, 0, fmt.Errorf("invalid resolution %q. Expected e.g. 720p, 1080p, 1440p, 4k, or 1920x1080", res)
}

func BuildScaleFilter(resolution string) (string, error) {
	if resolution == "" {
		return "", nil
	}
	w, h, err := ParseResolution(resolution)
	if err != nil {
		return "", err
	}
	if w > 0 && h > 0 {
		return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,scale=trunc(iw/2)*2:trunc(ih/2)*2", w, h), nil
	}
	return fmt.Sprintf("scale=-2:%d", h), nil
}

func BuildOptimizeArgs(input, output, resolution string, crf int) []string {
	if crf <= 0 {
		crf = 24
	}
	filter, _ := BuildScaleFilter(resolution)
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", input,
	}
	if filter != "" {
		args = append(args, "-vf", filter)
	}
	args = append(args,
		"-c:v", "libx264",
		"-crf", strconv.Itoa(crf),
		"-preset", "fast",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		output,
	)
	return args
}

type JobStatus string

const (
	StatusQueued     JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

type OptimizeJob struct {
	ID          string    `json:"id"`
	Input       string    `json:"input"`
	Output      string    `json:"output"`
	Resolution  string    `json:"resolution,omitempty"`
	CRF         int       `json:"crf,omitempty"`
	Status      JobStatus `json:"status"`
	CreatedAt   string    `json:"createdAt"`
	StartedAt   string    `json:"startedAt,omitempty"`
	CompletedAt string    `json:"completedAt,omitempty"`
	Error       string    `json:"error,omitempty"`
	InputBytes  int64     `json:"inputBytes,omitempty"`
	OutputBytes int64     `json:"outputBytes,omitempty"`
}

type OptimizeQueueData struct {
	WorkerPID int           `json:"workerPid,omitempty"`
	Jobs      []OptimizeJob `json:"jobs"`
}

func ReadQueue(queuePath string) OptimizeQueueData {
	queueMu.Lock()
	defer queueMu.Unlock()
	b, err := os.ReadFile(queuePath)
	if err != nil {
		return OptimizeQueueData{Jobs: []OptimizeJob{}}
	}
	var data OptimizeQueueData
	if err := json.Unmarshal(b, &data); err != nil {
		return OptimizeQueueData{Jobs: []OptimizeJob{}}
	}
	if data.Jobs == nil {
		data.Jobs = []OptimizeJob{}
	}
	return data
}

func WriteQueue(queuePath string, data OptimizeQueueData) error {
	queueMu.Lock()
	defer queueMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(queuePath), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", queuePath, os.Getpid())
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, queuePath)
}

func EnqueueJob(queuePath, input, output, res string, crf int, inBytes int64) OptimizeJob {
	data := ReadQueue(queuePath)
	job := OptimizeJob{
		ID:         fmt.Sprintf("opt_%d_%x", time.Now().UnixMilli(), rand.Int31()&0xffff),
		Input:      input,
		Output:     output,
		Resolution: res,
		CRF:        crf,
		Status:     StatusQueued,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		InputBytes: inBytes,
	}
	data.Jobs = append(data.Jobs, job)
	_ = WriteQueue(queuePath, data)
	return job
}

func UpdateJob(queuePath, id string, patch func(*OptimizeJob)) {
	data := ReadQueue(queuePath)
	for i := range data.Jobs {
		if data.Jobs[i].ID == id {
			patch(&data.Jobs[i])
			_ = WriteQueue(queuePath, data)
			return
		}
	}
}

func GetNextQueuedJob(queuePath string) *OptimizeJob {
	data := ReadQueue(queuePath)
	for _, j := range data.Jobs {
		if j.Status == StatusQueued {
			return &j
		}
	}
	return nil
}

func RecoverStaleJobs(queuePath string) {
	data := ReadQueue(queuePath)
	changed := false
	for i := range data.Jobs {
		if data.Jobs[i].Status == StatusProcessing {
			data.Jobs[i].Status = StatusQueued
			changed = true
		}
	}
	if changed {
		_ = WriteQueue(queuePath, data)
	}
}

func ClearQueue(queuePath string) {
	data := ReadQueue(queuePath)
	var remaining []OptimizeJob
	for _, j := range data.Jobs {
		if j.Status == StatusQueued || j.Status == StatusProcessing {
			remaining = append(remaining, j)
		}
	}
	data.Jobs = remaining
	_ = WriteQueue(queuePath, data)
}

func ProcessQueue(ctx context.Context, queuePath string, r runner.Runner, log func(string)) (int, error) {
	processed := 0
	RecoverStaleJobs(queuePath)

	for {
		select {
		case <-ctx.Done():
			return processed, ctx.Err()
		default:
		}

		job := GetNextQueuedJob(queuePath)
		if job == nil {
			break
		}

		UpdateJob(queuePath, job.ID, func(j *OptimizeJob) {
			j.Status = StatusProcessing
			j.StartedAt = time.Now().UTC().Format(time.RFC3339)
		})

		args := BuildOptimizeArgs(job.Input, job.Output, job.Resolution, job.CRF)
		cmd := append([]string{"ffmpeg"}, args...)
		if log != nil {
			log(fmt.Sprintf("optimizing %s -> %s", job.Input, job.Output))
		}

		_, stderr, code, _ := r.Run(ctx, cmd)
		if code == 0 {
			var outBytes int64
			if info, err := os.Stat(job.Output); err == nil {
				outBytes = info.Size()
			}
			UpdateJob(queuePath, job.ID, func(j *OptimizeJob) {
				j.Status = StatusCompleted
				j.CompletedAt = time.Now().UTC().Format(time.RFC3339)
				j.OutputBytes = outBytes
			})
			if log != nil {
				log(fmt.Sprintf("completed %s: %s", job.ID, job.Output))
			}
		} else {
			UpdateJob(queuePath, job.ID, func(j *OptimizeJob) {
				j.Status = StatusFailed
				j.CompletedAt = time.Now().UTC().Format(time.RFC3339)
				j.Error = strings.TrimSpace(stderr)
			})
			if log != nil {
				log(fmt.Sprintf("failed %s: %s", job.ID, stderr))
			}
		}
		processed++
	}
	return processed, nil
}
