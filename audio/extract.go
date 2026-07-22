// Package audio extracts and preprocesses audio from a video (or audio) file
// via ffmpeg into a transcriber-ready WAV. It has no dependency on frames or
// on any transcription backend.
package audio

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/bizshuk/video-utils/ffmpegutil"
)

// Options controls the extracted WAV's format.
type Options struct {
	// SampleRateHz defaults to 16000 — the rate nearly all ASR models
	// (whisper included) expect; extracting at the source rate would force
	// every transcriber backend to resample itself.
	SampleRateHz int
	// Channels defaults to 1 (mono). ASR models are trained on mono; stereo
	// input adds no signal for transcription. Set to 2 to keep stereo.
	Channels int
}

const (
	DefaultSampleRateHz = 16000
	DefaultChannels     = 1
)

// Extract demuxes videoPath's audio track into outPath (a .wav file; parent
// dir must already exist) and returns outPath on success.
func Extract(ctx context.Context, videoPath, outPath string, opts Options) (string, error) {
	return renderWAV(ctx, videoPath, outPath, opts, "", "extract audio")
}

func renderWAV(ctx context.Context, mediaPath, outPath string, opts Options, filter, operation string) (string, error) {
	if err := ffmpegutil.CheckAvailable(); err != nil {
		return "", err
	}
	if opts.SampleRateHz == 0 {
		opts.SampleRateHz = DefaultSampleRateHz
	}
	if opts.Channels == 0 {
		opts.Channels = DefaultChannels
	}

	args := []string{
		"-y", "-i", mediaPath,
		"-vn", // drop video stream — audio only
	}
	if filter != "" {
		args = append(args, "-af", filter)
	}
	args = append(args,
		"-ar", fmt.Sprintf("%d", opts.SampleRateHz),
		"-ac", fmt.Sprintf("%d", opts.Channels),
		"-c:a", "pcm_s16le",
		outPath,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg %s: %w: %s", operation, err, out)
	}
	return outPath, nil
}
