// Package segment splits media into fixed-duration chunks via ffmpeg.
package segment

import (
	"fmt"
	"path/filepath"
	"time"
)

// DefaultDuration is the per-segment length when Options.Duration is zero.
const DefaultDuration = 5 * time.Minute

// Options controls how media is split into fixed-length segments.
type Options struct {
	// Duration is the length of each segment. The final segment may be
	// shorter when the source window does not divide evenly. Zero uses
	// DefaultDuration. Duration takes priority over To when both would
	// constrain segment length: each chunk is Duration long, and To only
	// bounds the overall source window.
	Duration time.Duration
	// From is the start offset in the source media. Zero means the beginning.
	From time.Duration
	// To is the exclusive end offset in the source media. Zero means the end
	// of the media. Values past the media end are clamped.
	To time.Duration
}

// Segment is one emitted chunk, ordered from the start of the source window.
type Segment struct {
	Path     string
	Index    int
	Start    time.Duration
	Duration time.Duration
}

// Paths returns the on-disk path of each segment in order.
func Paths(segments []Segment) []string {
	paths := make([]string, 0, len(segments))
	for _, segment := range segments {
		paths = append(paths, segment.Path)
	}
	return paths
}

type timeRange struct {
	start    time.Duration
	duration time.Duration
}

func normalizeDuration(d time.Duration) (time.Duration, error) {
	if d == 0 {
		d = DefaultDuration
	}
	if d < 0 {
		return 0, fmt.Errorf("segment duration must be greater than zero: %s", d)
	}
	return d, nil
}

// resolveWindow returns the [start, end) source window to cut.
// From defaults to 0. To of 0 means the media end. Duration is applied
// separately as the per-segment chunk size and is the higher-priority
// control for how content is sliced; From/To only bound the window.
func resolveWindow(total, from, to time.Duration) (start, end time.Duration, err error) {
	if total <= 0 {
		return 0, 0, fmt.Errorf("media duration must be greater than zero: %s", total)
	}
	if from < 0 {
		return 0, 0, fmt.Errorf("from must be >= 0: %s", from)
	}
	if to < 0 {
		return 0, 0, fmt.Errorf("to must be >= 0: %s", to)
	}
	if from >= total {
		return 0, 0, fmt.Errorf("from %s is at or past media end %s", from, total)
	}

	start = from
	end = total
	if to > 0 {
		end = to
		if end > total {
			end = total
		}
	}
	if start >= end {
		return 0, 0, fmt.Errorf("empty cut window: from %s, to %s (media end %s)", from, end, total)
	}
	return start, end, nil
}

// planSegmentRanges builds absolute source ranges for opts over media of length total.
// Duration (chunk size) is the higher-priority slicing control; From/To bound the window.
func planSegmentRanges(total time.Duration, opts Options) ([]timeRange, error) {
	chunk, err := normalizeDuration(opts.Duration)
	if err != nil {
		return nil, err
	}
	start, end, err := resolveWindow(total, opts.From, opts.To)
	if err != nil {
		return nil, err
	}
	ranges := planRanges(end-start, chunk)
	for i := range ranges {
		ranges[i].start += start
	}
	return ranges, nil
}

// planRanges covers [0, total) with non-overlapping chunks of length chunk.
// total must be positive; the final range may be shorter than chunk.
func planRanges(total, chunk time.Duration) []timeRange {
	if total <= 0 || chunk <= 0 {
		return nil
	}

	var ranges []timeRange
	for start := time.Duration(0); start < total; start += chunk {
		length := chunk
		if remaining := total - start; remaining < length {
			length = remaining
		}
		ranges = append(ranges, timeRange{start: start, duration: length})
	}
	return ranges
}

func segmentPath(outDir, prefix string, index int, ext string) string {
	return filepath.Join(outDir, fmt.Sprintf("%s-%06d%s", prefix, index+1, ext))
}

func formatSeconds(d time.Duration) string {
	return fmt.Sprintf("%.3f", d.Seconds())
}
