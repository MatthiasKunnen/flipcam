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

func addInterfaceFlag(cmd *cobra.Command, stringVar *string) {
	cmd.Flags().StringVar(
		stringVar,
		"wireless-interface",
		"",
		"Sets the name of the wireless interface to use.",
	)
	err := cmd.MarkFlagRequired("wireless-interface")
	if err != nil {
		log.Fatal(err)
	}
}

func addIpv4Flag(cmd *cobra.Command, v *ipv4Flag) {
	cmd.Flags().Var(
		v,
		"ip",
		"Sets the ip for the AP. CIDR notation.",
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
