# Video Utilities

Standalone Go module for ffmpeg-based media preprocessing:
`github.com/bizshuk/video-utils`.

## Packages

- `audio`: extract an audio track or reduce steady white noise into a
  transcriber-ready WAV.
- `frames`: sample still frames with interval and scene-change options.
- `subtitles`: extract audio, run a pluggable transcriber, and write SRT.
- `segment`: cut media into fixed-duration audio WAV or video segments,
  with optional `--from` / `--to` source window bounds.
- `ffmpegutil`: check ffmpeg availability and probe media duration.
- `cmd`: reusable Cobra command tree rooted at the package-level `VideoCmd`.

The module does not import `github.com/bizshuk/agentsdk`.

## Command usage

> [!NOTE]
> This module provides reusable Cobra commands; it does not build a standalone
> executable. A host CLI such as `videoutils` registers `cmd.VideoCmd`:

```go
import videocmd "github.com/bizshuk/video-utils/cmd"

rootCmd.AddCommand(videocmd.VideoCmd)
```

The composed command exposes these stages:

```text
videoutils video
├── audio <video>       # stdout: generated WAV path
├── denoise <media>     # stdout: generated denoised WAV path
├── frames <video>      # stdout: one generated still path per line
├── subtitles <video>   # stdout: generated SRT path
├── cut-audio <media>   # stdout: one generated WAV segment path per line
└── cut-video <video>   # stdout: one generated video segment path per line
```

Example usage:

```bash
mkdir -p output

videoutils video audio input.mp4 --out output/audio.wav
videoutils video denoise input.mp4 --out output/denoised.wav
videoutils video frames input.mp4 --out output/frames --interval 2s
videoutils video cut-audio input.mp4 --out output/audio-segments --duration 5m
videoutils video cut-video input.mp4 --out output/video-segments --duration 5m
# Optional --from/--to bound the source window; --duration still owns segment length.
videoutils video cut-audio input.mp4 --out output/audio-segments --from 1m --to 20m --duration 5m
videoutils video cut-video input.mp4 --out output/video-segments --from 30s --to 10m --duration 2m
videoutils video subtitles input.mp4 \
  --engine whisper \
  --whisper-bin /path/to/whisper-cli \
  --whisper-model /path/to/ggml-model.bin \
  --out output/subtitles.srt
```

Successful stdout is path-only, so callers can capture or pipe the generated
paths directly. Informational summaries and warnings are written to stderr:

```bash
audio_path="$(videoutils video audio input.mp4 --out output/audio.wav)"
videoutils video frames input.mp4 --out output/frames > output/frame-paths.txt
```

## White-noise reduction

Use the reusable Go API:

```go
_, err := audio.ReduceWhiteNoise(ctx, "input.mp4", "denoised.wav", audio.WhiteNoiseOptions{})
```

Or the composed `videoutils` command:

```bash
videoutils video denoise input.mp4 --out denoised.wav
```

The defaults reduce white noise by `12 dB` with an estimated noise floor of
`-50 dB`. Tune them with `--reduction-db` and `--noise-floor-db`.

## Requirements

- Go `1.26+`
- `ffmpeg` and `ffprobe` on `PATH`
- An FFmpeg build containing the `afftdn` audio filter for white-noise reduction
- Optional Whisper.cpp or Qwen3/MLX runtime for real transcription

## Development

```bash
go test ./... -count=1 -timeout=120s
```
