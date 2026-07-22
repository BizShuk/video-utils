package cmd

import (
	"fmt"
	"time"

	"github.com/bizshuk/video-utils/ffmpegutil"
	"github.com/bizshuk/video-utils/frames"
	"github.com/spf13/cobra"
)

var (
	framesOutputDir      string
	framesInterval       time.Duration
	framesSceneThreshold float64
	framesMaxFrames      int
)

// FramesCmd extracts still frames from a video.
var FramesCmd = &cobra.Command{
	Use:   "frames <video>",
	Short: "Extract still frames from a video into a directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runFrames,
}

func init() {
	FramesCmd.Flags().StringVar(&framesOutputDir, "out", "./frames-out", "output directory for extracted frames")
	FramesCmd.Flags().DurationVar(&framesInterval, "interval", frames.DefaultInterval, "sampling interval (e.g. 2s)")
	FramesCmd.Flags().Float64Var(&framesSceneThreshold, "scene-threshold", 0, "additionally sample on scene changes above this score (0..1)")
	FramesCmd.Flags().IntVar(&framesMaxFrames, "max-frames", 0, "cap on emitted frames (0 = unlimited)")
}

func runFrames(cmd *cobra.Command, args []string) error {
	if duration, err := ffmpegutil.Probe(cmd.Context(), args[0]); err == nil {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "source duration: %s\n", duration); err != nil {
			return fmt.Errorf("print source duration: %w", err)
		}
	}

	extractedFrames, err := frames.Extract(cmd.Context(), args[0], framesOutputDir, frames.Options{
		Interval:       framesInterval,
		SceneThreshold: framesSceneThreshold,
		MaxFrames:      framesMaxFrames,
	})
	if err != nil {
		return fmt.Errorf("extract frames: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "extracted %d frame(s)\n", len(extractedFrames)); err != nil {
		return fmt.Errorf("print frame count: %w", err)
	}

	paths := make([]string, 0, len(extractedFrames))
	for _, frame := range extractedFrames {
		paths = append(paths, frame.Path)
	}
	return writeOutputPaths(cmd, paths...)
}
