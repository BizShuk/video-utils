package audio

import (
	"context"
	"fmt"
	"math"
)

const (
	// DefaultNoiseReductionDB is the amount of white-noise attenuation used
	// when WhiteNoiseOptions.ReductionDB is zero.
	DefaultNoiseReductionDB = 12.0
	// DefaultNoiseFloorDB is the assumed white-noise floor used when
	// WhiteNoiseOptions.NoiseFloorDB is zero.
	DefaultNoiseFloorDB = -50.0

	minNoiseReductionDB = 0.01
	maxNoiseReductionDB = 97.0
	minNoiseFloorDB     = -80.0
	maxNoiseFloorDB     = -20.0
)

// WhiteNoiseOptions controls white-noise reduction and the output WAV format.
type WhiteNoiseOptions struct {
	// SampleRateHz defaults to DefaultSampleRateHz.
	SampleRateHz int
	// Channels defaults to DefaultChannels.
	Channels int
	// ReductionDB is the requested attenuation in dB. It defaults to
	// DefaultNoiseReductionDB and must be between 0.01 and 97.
	ReductionDB float64
	// NoiseFloorDB is the measured or estimated noise floor in dB. It defaults
	// to DefaultNoiseFloorDB and must be between -80 and -20.
	NoiseFloorDB float64
}

// ReduceWhiteNoise extracts mediaPath's audio, reduces steady white noise via
// ffmpeg's FFT denoiser, and writes a PCM WAV to outPath. The parent directory
// of outPath must already exist.
func ReduceWhiteNoise(ctx context.Context, mediaPath, outPath string, opts WhiteNoiseOptions) (string, error) {
	normalized, err := normalizeWhiteNoiseOptions(opts)
	if err != nil {
		return "", err
	}

	return renderWAV(ctx, mediaPath, outPath, Options{
		SampleRateHz: normalized.SampleRateHz,
		Channels:     normalized.Channels,
	}, whiteNoiseFilter(normalized), "reduce white noise")
}

func normalizeWhiteNoiseOptions(opts WhiteNoiseOptions) (WhiteNoiseOptions, error) {
	if opts.SampleRateHz == 0 {
		opts.SampleRateHz = DefaultSampleRateHz
	}
	if opts.Channels == 0 {
		opts.Channels = DefaultChannels
	}
	if opts.ReductionDB == 0 {
		opts.ReductionDB = DefaultNoiseReductionDB
	}
	if opts.NoiseFloorDB == 0 {
		opts.NoiseFloorDB = DefaultNoiseFloorDB
	}

	if opts.SampleRateHz < 0 {
		return WhiteNoiseOptions{}, fmt.Errorf("sample rate must be greater than zero: %d", opts.SampleRateHz)
	}
	if opts.Channels < 0 {
		return WhiteNoiseOptions{}, fmt.Errorf("channels must be greater than zero: %d", opts.Channels)
	}
	if !isFinite(opts.ReductionDB) || opts.ReductionDB < minNoiseReductionDB || opts.ReductionDB > maxNoiseReductionDB {
		return WhiteNoiseOptions{}, fmt.Errorf("noise reduction must be between %.2f and %.0f dB: %g", minNoiseReductionDB, maxNoiseReductionDB, opts.ReductionDB)
	}
	if !isFinite(opts.NoiseFloorDB) || opts.NoiseFloorDB < minNoiseFloorDB || opts.NoiseFloorDB > maxNoiseFloorDB {
		return WhiteNoiseOptions{}, fmt.Errorf("noise floor must be between %.0f and %.0f dB: %g", minNoiseFloorDB, maxNoiseFloorDB, opts.NoiseFloorDB)
	}

	return opts, nil
}

func whiteNoiseFilter(opts WhiteNoiseOptions) string {
	return fmt.Sprintf(
		"afftdn=noise_reduction=%g:noise_floor=%g:noise_type=white",
		opts.ReductionDB,
		opts.NoiseFloorDB,
	)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
