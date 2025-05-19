package flipcam

import (
	"github.com/spf13/cobra"
	"log"
)

func addHlsOutputDirFlag(cmd *cobra.Command, stringVar *string) {
	cmd.Flags().StringVar(
		stringVar,
		"hls-output-dir",
		"",
		"Sets the directory where the HLS segments and playlists are stored.",
	)
	err := cmd.MarkFlagRequired("hls-output-dir")
	if err != nil {
		log.Fatal(err)
	}
}

func addHlsUrlPathPrefixFlag(cmd *cobra.Command, stringVar *string) {
	cmd.Flags().StringVar(
		stringVar,
		"hls-url-path-prefix",
		"/camera",
		"Sets the path prefix for the playlist URL.",
	)
}

func addUiPortFlag(cmd *cobra.Command, stringVar *string) {
	cmd.Flags().StringVar(
		stringVar,
		"ui-port",
		":3000",
		"Sets the port for the UI interface in the :port format.",
	)
}
