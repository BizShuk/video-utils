package cmd

import (
	"fmt"
	"time"

	"github.com/bizshuk/video-utils/audio"
	"github.com/bizshuk/video-utils/ffmpegutil"
	"github.com/bizshuk/video-utils/segment"
	"github.com/spf13/cobra"
)

var (
	cutAudioOutputDir    string
	cutAudioDuration     time.Duration
	cutAudioFrom         time.Duration
	cutAudioTo           time.Duration
	cutAudioSampleRateHz int
	cutAudioChannels     int
)

// CutAudioCmd splits media audio into fixed-duration WAV segments.
var CutAudioCmd = &cobra.Command{
	Use:   "cut-audio <media>",
	Short: "Cut media audio into fixed-duration WAV segments",
	Args:  cobra.ExactArgs(1),
	RunE:  runCutAudio,
}

func init() {
	CutAudioCmd.Flags().StringVar(&cutAudioOutputDir, "out", "./audio-segments", "output directory for audio segments")
	CutAudioCmd.Flags().DurationVar(&cutAudioDuration, "duration", segment.DefaultDuration, "length of each segment (e.g. 5m); higher priority than --to for slicing")
	CutAudioCmd.Flags().DurationVar(&cutAudioFrom, "from", 0, "start offset in source (e.g. 1m30s); default is start")
	CutAudioCmd.Flags().DurationVar(&cutAudioTo, "to", 0, "exclusive end offset in source (e.g. 10m); default is end of media")
	CutAudioCmd.Flags().IntVar(&cutAudioSampleRateHz, "sample-rate", audio.DefaultSampleRateHz, "sample rate in Hz")
	CutAudioCmd.Flags().IntVar(&cutAudioChannels, "channels", audio.DefaultChannels, "number of audio channels (1=mono, 2=stereo)")
}

func runCutAudio(cmd *cobra.Command, args []string) error {
	if duration, err := ffmpegutil.Probe(cmd.Context(), args[0]); err == nil {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "source duration: %s\n", duration); err != nil {
			return fmt.Errorf("print source duration: %w", err)
		}
	}

	segments, err := segment.SplitAudio(cmd.Context(), args[0], cutAudioOutputDir, segment.AudioOptions{
		Options: segment.Options{
			Duration: cutAudioDuration,
			From:     cutAudioFrom,
			To:       cutAudioTo,
		},
		SampleRateHz: cutAudioSampleRateHz,
		Channels:     cutAudioChannels,
	})
	if err != nil {
		return fmt.Errorf("cut audio: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d audio segment(s)\n", len(segments)); err != nil {
		return fmt.Errorf("print segment count: %w", err)
	}
	return writeOutputPaths(cmd, segment.Paths(segments)...)
}
