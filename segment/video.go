package segment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bizshuk/video-utils/ffmpegutil"
)

// VideoOptions controls fixed-duration video segment cutting.
type VideoOptions struct {
	Options
}

// SplitVideo cuts mediaPath into consecutive video segments of the configured
// duration and writes them under outDir (created if absent). Streams are
// copied when possible so cuts stay near the configured boundaries without a
// full re-encode. Options.From and Options.To bound the source window;
// Options.Duration is the per-segment length and takes priority for how the
// window is sliced.
func SplitVideo(ctx context.Context, mediaPath, outDir string, opts VideoOptions) ([]Segment, error) {
	if err := ffmpegutil.CheckAvailable(); err != nil {
		return nil, err
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

	ext := videoExtension(mediaPath)
	segments := make([]Segment, 0, len(ranges))
	for i, r := range ranges {
		outPath := segmentPath(outDir, "video", i, ext)
		// Input seeking (-ss before -i) keeps long files practical; -c copy
		// avoids a re-encode at the cost of keyframe-aligned starts.
		args := []string{
			"-y",
			"-ss", formatSeconds(r.start),
			"-i", mediaPath,
			"-t", formatSeconds(r.duration),
			"-c", "copy",
			"-avoid_negative_ts", "make_zero",
			outPath,
		}
		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("ffmpeg cut video segment %d: %w: %s", i+1, err, out)
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

func videoExtension(mediaPath string) string {
	ext := strings.ToLower(filepath.Ext(mediaPath))
	if ext == "" {
		return ".mp4"
	}
	return ext
}
