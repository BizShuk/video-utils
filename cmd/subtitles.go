package cmd

import (
	"fmt"
	"time"

	"github.com/bizshuk/video-utils/subtitles"
	"github.com/spf13/cobra"
)

const defaultQwenScript = "utils/video/subtitles/pyasr/qwen_transcribe.py"

var (
	subtitlesOutputPath string
	subtitlesWorkDir    string
	subtitlesKeepAudio  bool
	subtitlesEngine     string
	whisperBin          string
	whisperModel        string
	whisperLanguage     string
	qwenScript          string
	qwenModel           string
	qwenLanguage        string
	qwenChunkDuration   time.Duration
)

// SubtitlesCmd extracts audio and transcribes it into an SRT file.
var SubtitlesCmd = &cobra.Command{
	Use:   "subtitles <video>",
	Short: "Extract audio and transcribe it into a .srt subtitle file",
	Args:  cobra.ExactArgs(1),
	RunE:  runSubtitles,
}

func init() {
	SubtitlesCmd.Flags().StringVar(&subtitlesOutputPath, "out", "./out.srt", "output .srt path")
	SubtitlesCmd.Flags().StringVar(&subtitlesWorkDir, "work-dir", "./subtitles-work", "scratch dir for the intermediate audio track")
	SubtitlesCmd.Flags().BoolVar(&subtitlesKeepAudio, "keep-audio", false, "keep the intermediate .wav after transcription")
	SubtitlesCmd.Flags().StringVar(&subtitlesEngine, "engine", "noop", "transcription engine: whisper | qwen3 | noop")
	SubtitlesCmd.Flags().StringVar(&whisperBin, "whisper-bin", "", "path to whisper.cpp whisper-cli/main binary (--engine whisper)")
	SubtitlesCmd.Flags().StringVar(&whisperModel, "whisper-model", "", "path to a ggml whisper model file (--engine whisper)")
	SubtitlesCmd.Flags().StringVar(&whisperLanguage, "whisper-lang", "auto", "whisper language code, or auto (--engine whisper)")
	SubtitlesCmd.Flags().StringVar(&qwenScript, "qwen-script", defaultQwenScript, "path to qwen_transcribe.py (--engine qwen3)")
	SubtitlesCmd.Flags().StringVar(&qwenModel, "qwen-model", "", "mlx-community model id, empty = wrapper script default (--engine qwen3)")
	SubtitlesCmd.Flags().StringVar(&qwenLanguage, "qwen-lang", "", "language code, empty = auto-detect mixed EN/ZH (--engine qwen3)")
	SubtitlesCmd.Flags().DurationVar(&qwenChunkDuration, "qwen-chunk-duration", 10*time.Second, "subtitle-cue chunk length (--engine qwen3)")
}

func runSubtitles(cmd *cobra.Command, args []string) error {
	transcriber, warning, err := selectedTranscriber()
	if err != nil {
		return err
	}
	if warning != "" {
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), warning); err != nil {
			return fmt.Errorf("print transcriber warning: %w", err)
		}
	}

	segments, err := subtitles.Generate(cmd.Context(), args[0], subtitlesWorkDir, transcriber, subtitlesKeepAudio)
	if err != nil {
		return fmt.Errorf("generate subtitles: %w", err)
	}
	if err := subtitles.WriteSRT(segments, subtitlesOutputPath); err != nil {
		return fmt.Errorf("write subtitles: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d segment(s)\n", len(segments)); err != nil {
		return fmt.Errorf("print subtitle count: %w", err)
	}
	return writeOutputPaths(cmd, subtitlesOutputPath)
}

func selectedTranscriber() (subtitles.Transcriber, string, error) {
	switch subtitlesEngine {
	case "whisper":
		if whisperBin == "" || whisperModel == "" {
			return nil, "", fmt.Errorf("--engine whisper requires --whisper-bin and --whisper-model")
		}
		return subtitles.WhisperCPPTranscriber{
			BinPath:   whisperBin,
			ModelPath: whisperModel,
			Language:  whisperLanguage,
		}, "", nil
	case "qwen3":
		return subtitles.QwenMLXTranscriber{
			ScriptPath:    qwenScript,
			Model:         qwenModel,
			Language:      qwenLanguage,
			ChunkDuration: qwenChunkDuration,
		}, "", nil
	case "noop":
		return subtitles.NoopTranscriber{}, "warning: --engine noop produces 0 segments; pass --engine whisper or --engine qwen3 for real transcription", nil
	default:
		return nil, "", fmt.Errorf("unknown --engine %q: want whisper | qwen3 | noop", subtitlesEngine)
	}
}
