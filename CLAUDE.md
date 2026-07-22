# CLAUDE.md — video utilities technical context

`utils/video` is an independently versionable Go module for media preprocessing.
Its module path is `github.com/bizshuk/video-utils` and it uses Go
`1.26.0`.

## Structure

```text
utils/video/
├── go.mod
├── audio/       # audio extraction and white-noise reduction to WAV
├── frames/      # interval and scene-change still-frame sampling
├── subtitles/   # audio extraction, transcription, and SRT output
├── ffmpegutil/  # ffmpeg availability and ffprobe duration helpers
└── cmd/         # reusable package-level Cobra stage commands
```

Dependencies are one-way:

```text
audio ──> ffmpegutil
frames ──> ffmpegutil
subtitles ──> audio
cmd ──> audio, frames, subtitles, ffmpegutil
```

The module must not import `github.com/bizshuk/agentsdk`. The root SDK imports
only `cmd` to compose `auth-cli video`; the library packages do not depend on
the command package.

## Public command contract

`cmd.VideoCmd` is the reusable parent command. It registers the exported
package-level `cmd.AudioCmd`, `cmd.DenoiseCmd`, `cmd.FramesCmd`, and
`cmd.SubtitlesCmd` stage commands during package initialization. Flags bind in
`init()` and therefore retain Cobra state for the lifetime of the process.

On success, command stdout contains only output paths, one path per line:
`audio` and `denoise` print their WAV, `frames` prints every generated still,
and `subtitles` prints its SRT. Informational summaries and warnings use stderr.

## Runtime requirements

Library operations invoke `ffmpeg` and `ffprobe` from `PATH`; white-noise
reduction requires FFmpeg's `afftdn` audio filter. Real subtitles also require
either Whisper.cpp or the Qwen3/MLX wrapper runtime. Tests that need optional
system runtimes skip themselves when those runtimes are absent.

## Development

```bash
go mod tidy
go test ./... -count=1 -timeout=120s
go build ./...
```
