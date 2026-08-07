package segment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPlanRanges(t *testing.T) {
	tests := []struct {
		name  string
		total time.Duration
		chunk time.Duration
		want  []timeRange
	}{
		{
			name:  "even split",
			total: 10 * time.Second,
			chunk: 5 * time.Second,
			want: []timeRange{
				{start: 0, duration: 5 * time.Second},
				{start: 5 * time.Second, duration: 5 * time.Second},
			},
		},
		{
			name:  "short remainder",
			total: 7 * time.Second,
			chunk: 3 * time.Second,
			want: []timeRange{
				{start: 0, duration: 3 * time.Second},
				{start: 3 * time.Second, duration: 3 * time.Second},
				{start: 6 * time.Second, duration: 1 * time.Second},
			},
		},
		{
			name:  "single segment when shorter than chunk",
			total: 2 * time.Second,
			chunk: 5 * time.Second,
			want: []timeRange{
				{start: 0, duration: 2 * time.Second},
			},
		},
		{
			name:  "empty on zero total",
			total: 0,
			chunk: time.Second,
			want:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := planRanges(test.total, test.chunk)
			if len(got) != len(test.want) {
				t.Fatalf("len = %d, want %d (%v)", len(got), len(test.want), got)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Errorf("range[%d] = %+v, want %+v", i, got[i], test.want[i])
				}
			}
		})
	}
}

func TestNormalizeDuration(t *testing.T) {
	got, err := normalizeDuration(0)
	if err != nil {
		t.Fatalf("normalizeDuration(0): %v", err)
	}
	if got != DefaultDuration {
		t.Errorf("default duration = %s, want %s", got, DefaultDuration)
	}

	if _, err := normalizeDuration(-time.Second); err == nil {
		t.Fatal("expected error for negative duration")
	}
}

func TestResolveWindow(t *testing.T) {
	const total = 10 * time.Second
	tests := []struct {
		name      string
		from, to  time.Duration
		wantStart time.Duration
		wantEnd   time.Duration
		wantErr   bool
	}{
		{name: "full media", wantStart: 0, wantEnd: total},
		{name: "from only", from: 3 * time.Second, wantStart: 3 * time.Second, wantEnd: total},
		{name: "to only", to: 7 * time.Second, wantStart: 0, wantEnd: 7 * time.Second},
		{name: "from and to", from: 2 * time.Second, to: 8 * time.Second, wantStart: 2 * time.Second, wantEnd: 8 * time.Second},
		{name: "to past end clamps", to: 20 * time.Second, wantStart: 0, wantEnd: total},
		{name: "from at end", from: total, wantErr: true},
		{name: "from after end", from: total + time.Second, wantErr: true},
		{name: "empty window", from: 5 * time.Second, to: 5 * time.Second, wantErr: true},
		{name: "to before from", from: 6 * time.Second, to: 3 * time.Second, wantErr: true},
		{name: "negative from", from: -time.Second, wantErr: true},
		{name: "negative to", to: -time.Second, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end, err := resolveWindow(total, test.from, test.to)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got start=%s end=%s", start, end)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveWindow: %v", err)
			}
			if start != test.wantStart || end != test.wantEnd {
				t.Errorf("window = [%s, %s), want [%s, %s)", start, end, test.wantStart, test.wantEnd)
			}
		})
	}
}

func TestPlanSegmentRanges_FromToWithDurationPriority(t *testing.T) {
	// Duration is the chunk size (higher priority for slicing); From/To bound the window.
	// Media 10s, window [2s, 9s), chunk 3s → [2-5], [5-8], [8-9].
	got, err := planSegmentRanges(10*time.Second, Options{
		Duration: 3 * time.Second,
		From:     2 * time.Second,
		To:       9 * time.Second,
	})
	if err != nil {
		t.Fatalf("planSegmentRanges: %v", err)
	}
	want := []timeRange{
		{start: 2 * time.Second, duration: 3 * time.Second},
		{start: 5 * time.Second, duration: 3 * time.Second},
		{start: 8 * time.Second, duration: 1 * time.Second},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("range[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPlanSegmentRanges_DurationSlicesFullWindowWhenToOmitted(t *testing.T) {
	got, err := planSegmentRanges(5*time.Second, Options{
		Duration: 2 * time.Second,
		From:     time.Second,
	})
	if err != nil {
		t.Fatalf("planSegmentRanges: %v", err)
	}
	want := []timeRange{
		{start: time.Second, duration: 2 * time.Second},
		{start: 3 * time.Second, duration: 2 * time.Second},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("range[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSplitAudio(t *testing.T) {
	dir := t.TempDir()
	media := synthTestClip(t, dir, 5)
	outDir := filepath.Join(dir, "audio-segments")

	segments, err := SplitAudio(context.Background(), media, outDir, AudioOptions{
		Options: Options{Duration: 2 * time.Second},
	})
	if err != nil {
		t.Fatalf("SplitAudio: %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("segment count = %d, want 3", len(segments))
	}

	for i, segment := range segments {
		if _, err := os.Stat(segment.Path); err != nil {
			t.Errorf("segment %d missing: %v", i, err)
		}
		if segment.Index != i {
			t.Errorf("segment %d index = %d", i, segment.Index)
		}
	}
	if segments[2].Duration != time.Second {
		t.Errorf("final segment duration = %s, want 1s", segments[2].Duration)
	}
}

func TestSplitVideo(t *testing.T) {
	dir := t.TempDir()
	media := synthTestClip(t, dir, 5)
	outDir := filepath.Join(dir, "video-segments")

	segments, err := SplitVideo(context.Background(), media, outDir, VideoOptions{
		Options: Options{Duration: 2 * time.Second},
	})
	if err != nil {
		t.Fatalf("SplitVideo: %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("segment count = %d, want 3", len(segments))
	}

	for i, segment := range segments {
		info, err := os.Stat(segment.Path)
		if err != nil {
			t.Errorf("segment %d missing: %v", i, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("segment %d is empty", i)
		}
		if filepath.Ext(segment.Path) != ".mp4" {
			t.Errorf("segment %d ext = %q, want .mp4", i, filepath.Ext(segment.Path))
		}
	}
}

func TestSplitAudio_FromToWindow(t *testing.T) {
	dir := t.TempDir()
	media := synthTestClip(t, dir, 6)
	outDir := filepath.Join(dir, "audio-window")

	// Window [1s, 5s) with 2s chunks → two full segments; duration owns slice size.
	segments, err := SplitAudio(context.Background(), media, outDir, AudioOptions{
		Options: Options{
			Duration: 2 * time.Second,
			From:     time.Second,
			To:       5 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("SplitAudio: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("segment count = %d, want 2", len(segments))
	}
	if segments[0].Start != time.Second || segments[0].Duration != 2*time.Second {
		t.Errorf("segment[0] = start %s dur %s, want start 1s dur 2s", segments[0].Start, segments[0].Duration)
	}
	if segments[1].Start != 3*time.Second || segments[1].Duration != 2*time.Second {
		t.Errorf("segment[1] = start %s dur %s, want start 3s dur 2s", segments[1].Start, segments[1].Duration)
	}
}

func TestSplitVideo_FromToWindow(t *testing.T) {
	dir := t.TempDir()
	media := synthTestClip(t, dir, 6)
	outDir := filepath.Join(dir, "video-window")

	segments, err := SplitVideo(context.Background(), media, outDir, VideoOptions{
		Options: Options{
			Duration: 2 * time.Second,
			From:     time.Second,
			To:       5 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("SplitVideo: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("segment count = %d, want 2", len(segments))
	}
	for i, segment := range segments {
		if _, err := os.Stat(segment.Path); err != nil {
			t.Errorf("segment %d missing: %v", i, err)
		}
	}
	if segments[0].Start != time.Second {
		t.Errorf("first start = %s, want 1s", segments[0].Start)
	}
}

func TestSplitAudio_MissingSource(t *testing.T) {
	dir := t.TempDir()
	_, err := SplitAudio(context.Background(), filepath.Join(dir, "nope.mp4"), filepath.Join(dir, "out"), AudioOptions{
		Options: Options{Duration: time.Second},
	})
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}

func TestSplitVideo_MissingSource(t *testing.T) {
	dir := t.TempDir()
	_, err := SplitVideo(context.Background(), filepath.Join(dir, "nope.mp4"), filepath.Join(dir, "out"), VideoOptions{
		Options: Options{Duration: time.Second},
	})
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}

func TestPaths(t *testing.T) {
	got := Paths([]Segment{
		{Path: "a.wav"},
		{Path: "b.wav"},
	})
	if len(got) != 2 || got[0] != "a.wav" || got[1] != "b.wav" {
		t.Fatalf("Paths = %v", got)
	}
}

// synthTestClip renders a short clip with video + audio via lavfi.
func synthTestClip(t *testing.T, dir string, seconds int) string {
	t.Helper()
	path := filepath.Join(dir, "synth.mp4")
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", fmt.Sprintf("testsrc=duration=%d:size=64x64:rate=10", seconds),
		"-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=440:duration=%d", seconds),
		"-pix_fmt", "yuv420p",
		"-shortest",
		path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg unavailable or failed to synth test clip: %v: %s", err, out)
	}
	return path
}
