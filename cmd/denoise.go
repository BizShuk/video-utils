package cmd

import (
	"fmt"

	"github.com/bizshuk/video-utils/audio"
	"github.com/spf13/cobra"
)

var (
	denoiseOutputPath   string
	denoiseSampleRateHz int
	denoiseChannels     int
	denoiseReductionDB  float64
	denoiseNoiseFloorDB float64
)

// DenoiseCmd extracts audio and reduces steady white noise.
var DenoiseCmd = &cobra.Command{
	Use:   "denoise <media>",
	Short: "Extract audio and reduce steady white noise",
	Args:  cobra.ExactArgs(1),
	RunE:  runDenoise,
}

func init() {
	DenoiseCmd.Flags().StringVar(&denoiseOutputPath, "out", "./denoised.wav", "output .wav path")
	DenoiseCmd.Flags().IntVar(&denoiseSampleRateHz, "sample-rate", audio.DefaultSampleRateHz, "sample rate in Hz")
	DenoiseCmd.Flags().IntVar(&denoiseChannels, "channels", audio.DefaultChannels, "number of audio channels (1=mono, 2=stereo)")
	DenoiseCmd.Flags().Float64Var(&denoiseReductionDB, "reduction-db", audio.DefaultNoiseReductionDB, "white-noise attenuation in dB (0.01..97)")
	DenoiseCmd.Flags().Float64Var(&denoiseNoiseFloorDB, "noise-floor-db", audio.DefaultNoiseFloorDB, "estimated white-noise floor in dB (-80..-20)")
}

func runDenoise(cmd *cobra.Command, args []string) error {
	outputPath, err := audio.ReduceWhiteNoise(cmd.Context(), args[0], denoiseOutputPath, audio.WhiteNoiseOptions{
		SampleRateHz: denoiseSampleRateHz,
		Channels:     denoiseChannels,
		ReductionDB:  denoiseReductionDB,
		NoiseFloorDB: denoiseNoiseFloorDB,
	})
	if err != nil {
		return fmt.Errorf("reduce white noise: %w", err)
	}

	return writeOutputPaths(cmd, outputPath)
}
