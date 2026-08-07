package cmd

import (
	"fmt"
	"time"

	"github.com/bizshuk/video-utils/ffmpegutil"
	"github.com/bizshuk/video-utils/segment"
	"github.com/spf13/cobra"
)

var (
	cutVideoOutputDir string
	cutVideoDuration  time.Duration
	cutVideoFrom      time.Duration
	cutVideoTo        time.Duration
)

// CutVideoCmd splits a video into fixed-duration video segments.
var CutVideoCmd = &cobra.Command{
	Use:   "cut-video <video>",
	Short: "Cut a video into fixed-duration segments",
	Args:  cobra.ExactArgs(1),
	RunE:  runCutVideo,
}

func init() {
	CutVideoCmd.Flags().StringVar(&cutVideoOutputDir, "out", "./video-segments", "output directory for video segments")
	CutVideoCmd.Flags().DurationVar(&cutVideoDuration, "duration", segment.DefaultDuration, "length of each segment (e.g. 5m); higher priority than --to for slicing")
	CutVideoCmd.Flags().DurationVar(&cutVideoFrom, "from", 0, "start offset in source (e.g. 1m30s); default is start")
	CutVideoCmd.Flags().DurationVar(&cutVideoTo, "to", 0, "exclusive end offset in source (e.g. 10m); default is end of media")
}

func runCutVideo(cmd *cobra.Command, args []string) error {
	if duration, err := ffmpegutil.Probe(cmd.Context(), args[0]); err == nil {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "source duration: %s\n", duration); err != nil {
			return fmt.Errorf("print source duration: %w", err)
		}
	}

	segments, err := segment.SplitVideo(cmd.Context(), args[0], cutVideoOutputDir, segment.VideoOptions{
		Options: segment.Options{
			Duration: cutVideoDuration,
			From:     cutVideoFrom,
			To:       cutVideoTo,
		},
	})
	if err != nil {
		return fmt.Errorf("cut video: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d video segment(s)\n", len(segments)); err != nil {
		return fmt.Errorf("print segment count: %w", err)
	}
	return writeOutputPaths(cmd, segment.Paths(segments)...)
}
