package cmd

import (
	"fmt"

	"github.com/bizshuk/video-utils/audio"
	"github.com/spf13/cobra"
)

var (
	audioOutputPath   string
	audioSampleRateHz int
	audioChannels     int
)

// AudioCmd extracts a video's audio track into a WAV file.
var AudioCmd = &cobra.Command{
	Use:   "audio <video>",
	Short: "Extract audio track from a video into a WAV file",
	Args:  cobra.ExactArgs(1),
	RunE:  runAudio,
}

func init() {
	AudioCmd.Flags().StringVar(&audioOutputPath, "out", "./audio.wav", "output .wav path")
	AudioCmd.Flags().IntVar(&audioSampleRateHz, "sample-rate", audio.DefaultSampleRateHz, "sample rate in Hz")
	AudioCmd.Flags().IntVar(&audioChannels, "channels", audio.DefaultChannels, "number of audio channels (1=mono, 2=stereo)")
}

func runAudio(cmd *cobra.Command, args []string) error {
	outputPath, err := audio.Extract(cmd.Context(), args[0], audioOutputPath, audio.Options{
		SampleRateHz: audioSampleRateHz,
		Channels:     audioChannels,
	})
	if err != nil {
		return fmt.Errorf("extract audio: %w", err)
	}

	return writeOutputPaths(cmd, outputPath)
}
