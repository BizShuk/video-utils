package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestVideoCmdBuildsStageCommandTree(t *testing.T) {
	if VideoCmd.Use != "video" {
		t.Fatalf("Use = %q, want %q", VideoCmd.Use, "video")
	}

	tests := []struct {
		use  string
		want *cobra.Command
	}{
		{use: "audio <video>", want: AudioCmd},
		{use: "denoise <media>", want: DenoiseCmd},
		{use: "frames <video>", want: FramesCmd},
		{use: "subtitles <video>", want: SubtitlesCmd},
	}
	for _, test := range tests {
		t.Run(test.use, func(t *testing.T) {
			if got := findCommand(VideoCmd, test.use); got != test.want {
				t.Errorf("command %q = %p, want %p", test.use, got, test.want)
			}
		})
	}
}

func TestDenoiseCmdDefaults(t *testing.T) {
	want := map[string]string{
		"out":            "./denoised.wav",
		"sample-rate":    "16000",
		"channels":       "1",
		"reduction-db":   "12",
		"noise-floor-db": "-50",
	}
	for name, value := range want {
		flag := DenoiseCmd.Flags().Lookup(name)
		if flag == nil {
			t.Errorf("denoise --%s flag missing", name)
			continue
		}
		if got := flag.DefValue; got != value {
			t.Errorf("denoise --%s default = %q, want %q", name, got, value)
		}
	}
}

func TestStageCommandsPrintOnlyOutputPathsToStdout(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "ffmpeg"), `#!/bin/sh
last=""
for argument in "$@"; do
	last="$argument"
done
output=$(printf '%s\n' "$last" | sed 's/%06d/000001/')
mkdir -p "$(dirname "$output")"
: > "$output"
printf '%s\n' 'pts_time:0.000' >&2
`)
	writeExecutable(t, filepath.Join(binDir, "ffprobe"), `#!/bin/sh
printf '%s\n' '{"format":{"duration":"1.000"}}'
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	outputDir := t.TempDir()
	oldAudioOutputPath := audioOutputPath
	oldDenoiseOutputPath := denoiseOutputPath
	oldFramesOutputDir := framesOutputDir
	oldFramesInterval := framesInterval
	oldFramesSceneThreshold := framesSceneThreshold
	oldFramesMaxFrames := framesMaxFrames
	oldSubtitlesOutputPath := subtitlesOutputPath
	oldSubtitlesWorkDir := subtitlesWorkDir
	oldSubtitlesKeepAudio := subtitlesKeepAudio
	oldSubtitlesEngine := subtitlesEngine
	t.Cleanup(func() {
		audioOutputPath = oldAudioOutputPath
		denoiseOutputPath = oldDenoiseOutputPath
		framesOutputDir = oldFramesOutputDir
		framesInterval = oldFramesInterval
		framesSceneThreshold = oldFramesSceneThreshold
		framesMaxFrames = oldFramesMaxFrames
		subtitlesOutputPath = oldSubtitlesOutputPath
		subtitlesWorkDir = oldSubtitlesWorkDir
		subtitlesKeepAudio = oldSubtitlesKeepAudio
		subtitlesEngine = oldSubtitlesEngine
	})

	audioOutputPath = filepath.Join(outputDir, "audio.wav")
	denoiseOutputPath = filepath.Join(outputDir, "denoised.wav")
	framesOutputDir = filepath.Join(outputDir, "frames")
	framesInterval = time.Second
	framesSceneThreshold = 0
	framesMaxFrames = 0
	subtitlesOutputPath = filepath.Join(outputDir, "subtitles.srt")
	subtitlesWorkDir = filepath.Join(outputDir, "subtitles-work")
	subtitlesKeepAudio = false
	subtitlesEngine = "noop"

	tests := []struct {
		name string
		run  func(*cobra.Command, []string) error
		want []string
	}{
		{name: "audio", run: runAudio, want: []string{audioOutputPath}},
		{name: "denoise", run: runDenoise, want: []string{denoiseOutputPath}},
		{name: "frames", run: runFrames, want: []string{filepath.Join(framesOutputDir, "frame-000001.jpg")}},
		{name: "subtitles", run: runSubtitles, want: []string{subtitlesOutputPath}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			command := &cobra.Command{}
			command.SetContext(context.Background())
			command.SetOut(&stdout)
			command.SetErr(&stderr)

			if err := test.run(command, []string{"input.mp4"}); err != nil {
				t.Fatalf("run stage: %v", err)
			}

			want := strings.Join(test.want, "\n") + "\n"
			if got := stdout.String(); got != want {
				t.Errorf("stdout = %q, want path-only output %q", got, want)
			}
		})
	}
}

func TestWriteOutputPathsUsesCobraOutput(t *testing.T) {
	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)

	if err := writeOutputPaths(command, "audio.wav", "frame-000001.jpg"); err != nil {
		t.Fatalf("writeOutputPaths: %v", err)
	}

	const want = "audio.wav\nframe-000001.jpg\n"
	if got := output.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestWriteOutputPathsReturnsWriterError(t *testing.T) {
	command := &cobra.Command{}
	command.SetOut(errorWriter{})

	err := writeOutputPaths(command, "audio.wav")
	if err == nil || !strings.Contains(err.Error(), "print output path") {
		t.Fatalf("writeOutputPaths error = %v, want print context", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func findCommand(root *cobra.Command, use string) *cobra.Command {
	for _, child := range root.Commands() {
		if child.Use == use {
			return child
		}
	}
	return nil
}
