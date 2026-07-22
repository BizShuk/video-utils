package cmd

import "github.com/spf13/cobra"

// VideoCmd groups the media preprocessing stage commands.
var VideoCmd = &cobra.Command{
	Use:   "video",
	Short: "Video preprocessing utilities (audio, denoise, frames, subtitles)",
}

func init() {
	VideoCmd.AddCommand(AudioCmd, DenoiseCmd, FramesCmd, SubtitlesCmd)
}
