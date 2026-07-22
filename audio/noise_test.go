package audio

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReduceWhiteNoise_DefaultsToMono16kHzWav(t *testing.T) {
	dir := t.TempDir()
	mediaPath := synthTestClip(t, dir, 2)
	outPath := filepath.Join(dir, "denoised.wav")

	got, err := ReduceWhiteNoise(context.Background(), mediaPath, outPath, WhiteNoiseOptions{})
	if err != nil {
		t.Fatalf("ReduceWhiteNoise: %v", err)
	}
	if got != outPath {
		t.Errorf("returned path = %q, want %q", got, outPath)
	}

	sampleRate, channels := readWavFormat(t, outPath)
	if sampleRate != DefaultSampleRateHz {
		t.Errorf("sample rate = %d, want %d", sampleRate, DefaultSampleRateHz)
	}
	if channels != DefaultChannels {
		t.Errorf("channels = %d, want %d", channels, DefaultChannels)
	}
}

func TestReduceWhiteNoise_AttenuatesWhiteNoise(t *testing.T) {
	dir := t.TempDir()
	mediaPath := synthWhiteNoise(t, dir)
	outPath := filepath.Join(dir, "denoised.wav")

	_, err := ReduceWhiteNoise(context.Background(), mediaPath, outPath, WhiteNoiseOptions{
		ReductionDB:  30,
		NoiseFloorDB: -35,
	})
	if err != nil {
		t.Fatalf("ReduceWhiteNoise: %v", err)
	}

	inputRMS := wavPCM16RMS(t, mediaPath)
	outputRMS := wavPCM16RMS(t, outPath)
	if outputRMS >= inputRMS*0.5 {
		t.Fatalf("output RMS = %.2f, want less than half of input RMS %.2f", outputRMS, inputRMS)
	}
}

func TestNormalizeWhiteNoiseOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    WhiteNoiseOptions
		wantErr string
	}{
		{name: "defaults"},
		{name: "minimums", opts: WhiteNoiseOptions{ReductionDB: 0.01, NoiseFloorDB: -80}},
		{name: "maximums", opts: WhiteNoiseOptions{ReductionDB: 97, NoiseFloorDB: -20}},
		{name: "negative sample rate", opts: WhiteNoiseOptions{SampleRateHz: -1}, wantErr: "sample rate"},
		{name: "negative channels", opts: WhiteNoiseOptions{Channels: -1}, wantErr: "channels"},
		{name: "reduction too low", opts: WhiteNoiseOptions{ReductionDB: -1}, wantErr: "noise reduction"},
		{name: "reduction too high", opts: WhiteNoiseOptions{ReductionDB: 98}, wantErr: "noise reduction"},
		{name: "reduction not finite", opts: WhiteNoiseOptions{ReductionDB: math.NaN()}, wantErr: "noise reduction"},
		{name: "floor too low", opts: WhiteNoiseOptions{NoiseFloorDB: -81}, wantErr: "noise floor"},
		{name: "floor too high", opts: WhiteNoiseOptions{NoiseFloorDB: -19}, wantErr: "noise floor"},
		{name: "positive floor", opts: WhiteNoiseOptions{NoiseFloorDB: 1}, wantErr: "noise floor"},
		{name: "floor not finite", opts: WhiteNoiseOptions{NoiseFloorDB: math.Inf(-1)}, wantErr: "noise floor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeWhiteNoiseOptions(tt.opts)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeWhiteNoiseOptions: %v", err)
			}
			if got.SampleRateHz == 0 || got.Channels == 0 || got.ReductionDB == 0 || got.NoiseFloorDB == 0 {
				t.Fatalf("options not fully normalized: %+v", got)
			}
		})
	}
}

func TestWhiteNoiseFilter(t *testing.T) {
	got := whiteNoiseFilter(WhiteNoiseOptions{ReductionDB: 12.5, NoiseFloorDB: -42.5})
	want := "afftdn=noise_reduction=12.5:noise_floor=-42.5:noise_type=white"
	if got != want {
		t.Fatalf("whiteNoiseFilter = %q, want %q", got, want)
	}
}

func TestReduceWhiteNoiseRejectsInvalidOptionsBeforeRunningFFmpeg(t *testing.T) {
	_, err := ReduceWhiteNoise(context.Background(), "missing.mp4", "out.wav", WhiteNoiseOptions{
		ReductionDB: 98,
	})
	if err == nil || !strings.Contains(err.Error(), "noise reduction") {
		t.Fatalf("error = %v, want noise reduction validation error", err)
	}
}

func synthWhiteNoise(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "noise.wav")
	cmd := exec.Command(
		"ffmpeg", "-y",
		"-f", "lavfi",
		"-i", "anoisesrc=color=white:amplitude=0.01:duration=3:sample_rate=16000",
		"-c:a", "pcm_s16le",
		path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg unavailable or failed to synthesize white noise: %v: %s", err, out)
	}
	return path
}

func wavPCM16RMS(t *testing.T, path string) float64 {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wav: %v", err)
	}
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		t.Fatalf("not a WAV file: %s", path)
	}

	for offset := 12; offset+8 <= len(data); {
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkStart := offset + 8
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > len(data) {
			t.Fatalf("invalid WAV chunk at offset %d", offset)
		}
		if string(data[offset:offset+4]) == "data" {
			var sumSquares float64
			sampleCount := 0
			for i := chunkStart; i+2 <= chunkEnd; i += 2 {
				sample := float64(int16(binary.LittleEndian.Uint16(data[i : i+2])))
				sumSquares += sample * sample
				sampleCount++
			}
			if sampleCount == 0 {
				t.Fatal("WAV data chunk has no samples")
			}
			return math.Sqrt(sumSquares / float64(sampleCount))
		}
		offset = chunkEnd + chunkSize%2
	}

	t.Fatal("WAV data chunk not found")
	return 0
}
