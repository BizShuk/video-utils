package segment

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/bizshuk/video-utils/audio"
	"github.com/bizshuk/video-utils/ffmpegutil"
)

// AudioOptions controls fixed-duration audio segment extraction.
type AudioOptions struct {
	Options
	// SampleRateHz defaults to audio.DefaultSampleRateHz.
	SampleRateHz int
	// Channels defaults to audio.DefaultChannels.
	Channels int
}

// SplitAudio cuts mediaPath into consecutive WAV segments of the configured
// duration and writes them under outDir (created if absent). Options.From and
// Options.To bound the source window; Options.Duration is the per-segment
// length and takes priority for how the window is sliced.
func SplitAudio(ctx context.Context, mediaPath, outDir string, opts AudioOptions) ([]Segment, error) {
	if err := ffmpegutil.CheckAvailable(); err != nil {
		return nil, err
	}
	if opts.SampleRateHz == 0 {
		opts.SampleRateHz = audio.DefaultSampleRateHz
	}
	if opts.Channels == 0 {
		opts.Channels = audio.DefaultChannels
	}
	if opts.SampleRateHz < 0 {
		return nil, fmt.Errorf("sample rate must be greater than zero: %d", opts.SampleRateHz)
	}
	if opts.Channels < 0 {
		return nil, fmt.Errorf("channels must be greater than zero: %d", opts.Channels)
	}

	total, err := ffmpegutil.Probe(ctx, mediaPath)
	if err != nil {
		return nil, fmt.Errorf("probe media: %w", err)
	}

	ranges, err := planSegmentRanges(total, opts.Options)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create out dir: %w", err)
	}

	segments := make([]Segment, 0, len(ranges))
	for i, r := range ranges {
		outPath := segmentPath(outDir, "audio", i, ".wav")
		args := []string{
			"-y",
			"-ss", formatSeconds(r.start),
			"-i", mediaPath,
			"-t", formatSeconds(r.duration),
			"-vn",
			"-ar", fmt.Sprintf("%d", opts.SampleRateHz),
			"-ac", fmt.Sprintf("%d", opts.Channels),
			"-c:a", "pcm_s16le",
			outPath,
		}
		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("ffmpeg cut audio segment %d: %w: %s", i+1, err, out)
		}
		segments = append(segments, Segment{
			Path:     outPath,
			Index:    i,
			Start:    r.start,
			Duration: r.duration,
		})
	}
	return segments, nil
}
