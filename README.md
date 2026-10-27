# Goscribe

Zero-cloud screen recorder, Whisper transcription & AI meeting intelligence in pure Go.

Goscribe provides a Go CLI for local macOS recording, with native Swift capture helpers and external media and AI tools. Recordings default to `~/Movies/goscribe`; recording state and the optimization queue live in `~/.goscribe`.

**Privacy:** Recording and Whisper.cpp transcription are local. Optional Hermes meeting intelligence may send transcripts to a cloud model, depending on your Hermes configuration. Use a local model backend if you require zero-cloud operation.

## Architecture

- **ScreenCaptureKit** — native Swift helpers capture displays, windows, selected regions, and system audio.
- **FFmpeg** — captures microphone audio, provides an alternative screen capture engine, mixes/muxes audio and video, and optimizes recordings.
- **Whisper.cpp** — local speech-to-text architecture, with Go utilities for language selection, model lookup, and transcript formatting.
- **Hermes** — generates meeting notes from existing transcripts through the `hermes` CLI.

The current CLI implements recording and transcript-based notes generation; automatic transcription and the interactive live view are not yet wired into `start`. Some options shown by `goscribe --help` describe that intended workflow rather than currently accepted flags.

## Prerequisites

- **macOS 13+** for the project baseline. The current `native/capture.swift` ScreenCaptureKit helper targets **macOS 14+**; use macOS 14+ for that capture path.
- **Go 1.27.2**, as declared in `go.mod`.
- **FFmpeg**, available on `PATH`.
- **Xcode Command Line Tools**, including `swiftc`, for compiling native helpers.
- Screen Recording and Microphone permissions for your terminal or recording process, granted in macOS System Settings when prompted.
- For the AI workflow: Whisper.cpp with local model files, and Hermes configured with your chosen model backend. `notes` requires `hermes` on `PATH` or at `~/.local/bin/hermes`, plus an existing transcript.

Install the macOS tools:

```sh
xcode-select --install
brew install ffmpeg
```

Run capture commands from the repository root: native helper source paths are currently relative to the working directory.

## Build and test

```sh
make build          # Build bin/goscribe
make test           # Run go test -v ./...
make                # Test, then build
make install        # Install with go install ./cmd/goscribe
make clean          # Remove bin/ and clear the Go test cache
```

Use `./bin/goscribe` after building, or add your Go installation's binary directory (normally `$(go env GOPATH)/bin`) to `PATH` after `make install`.

## Quick start

```sh
./bin/goscribe devices
./bin/goscribe start --name "Team meeting"
./bin/goscribe status
./bin/goscribe stop
./bin/goscribe list
./bin/goscribe open
```

`start` launches recording processes in the background and returns; `stop` finalizes the recording.

## CLI reference

The commands below use `goscribe` for brevity. Put flags **before positional paths**: the current Go flag parser stops parsing flags at the first positional argument. Commands accepting an optional target use the latest recording in `~/Movies/goscribe` when omitted.

### `start`

```text
goscribe start [--name NAME] [--out FILE] [--audio-only]
               [--screen N] [--fps N] [--mic N] [--no-mic]
               [--no-system-audio] [--engine sck|ffmpeg]
               [--window ID_OR_NAME] [--area x,y,w,h] [--display ID]
```

Start screen/audio recording. FPS defaults to 30. Use `devices` to find screen and microphone indices, display IDs, and window IDs/names. Window, region, and display selection require the ScreenCaptureKit engine. `--audio-only` records audio without video; `--no-mic` and `--no-system-audio` disable their respective sources.

```sh
goscribe start --name "Demo" --fps 30
goscribe start --engine sck --window "Safari"
goscribe start --audio-only --out meeting.m4a
```

`--transcribe`, `--live`, `--notes`, `--draw`, `--detach`, `--lang`, `--role`, and `--context` are not currently accepted by `start`, despite appearing in the top-level help. The advertised `live` command is not implemented.

### `stop`

```text
goscribe stop [--notify]
```

Stop the active recording and finalize its media files. `--notify` posts a macOS notification when saved.

### `status`

```text
goscribe status [--json]
```

Show current recording status, optionally as JSON.

### `list`

```text
goscribe list
```

List recordings in `~/Movies/goscribe`.

### `open`

```text
goscribe open [--notes | --transcribe] [-R | --reveal] [TARGET]
```

Open a recording, its existing `.notes.md`, or its existing `.transcript.md` in the default app. If the requested sidecar does not exist, open the recording instead. `--reveal` shows the chosen file in Finder. `--transcribe` selects an existing transcript; it does not run transcription.

### `copy`

```text
goscribe copy [--transcribe] [TARGET]
```

Copy existing notes to the macOS clipboard, falling back to the transcript. `--transcribe` selects only the existing transcript.

### `last`

```text
goscribe last [--json]
```

Show the latest recording, optionally as JSON.

### `devices`

```text
goscribe devices [--json]
```

List available capture devices and sources, optionally as JSON. Uses FFmpeg and the native ScreenCaptureKit helper.

### `captions`

```text
goscribe captions
```

Follow the active recording's `.live.txt` caption file until recording stops or you press Ctrl-C. Requires a caption producer; the current `start` command does not generate live captions itself.

### `notes`

```text
goscribe notes [--lang LANGUAGE] [--name NAME] RECORDING_OR_TRANSCRIPT
```

Generate meeting notes through Hermes using an existing `.transcript.md` alongside the recording, or a transcript path directly. Accepts `.mp4`, `.m4a`, or `.transcript.md` targets. This command does not transcribe audio. Language may be a name or code, such as `english` or `en`.

```sh
goscribe notes --lang english --name "Team meeting" meeting.transcript.md
```

**Privacy:** Hermes may send transcript contents to a cloud model. Check its model configuration before using sensitive recordings.

### `optimize`

```text
goscribe optimize [--resolution RES | --res RES] [--crf N]
                  [--out FILE] [--wait | -w] [TARGET]
goscribe optimize [--status | --clear]
```

Enqueue FFmpeg optimization. Resolution examples: `720p`, `1080p`, `4k`. CRF defaults to 24. `--wait` processes the queue synchronously; without it, this command only enqueues work. `--status` shows the queue; `--clear` removes completed and failed jobs.

```sh
goscribe optimize --resolution 720p --crf 24 --wait recording.mp4
```

### `queue`

```text
goscribe queue [--clear] [--json]
```

Show optimization jobs, optionally as JSON. `--clear` removes completed and failed jobs.
