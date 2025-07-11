package flipcam

import (
	"github.com/spf13/cobra"
	"log"
	"net"
	"os"
	"path"
	"strings"
)

func addHlsOutputDirFlag(cmd *cobra.Command, stringVar *string) {
	cmd.Flags().StringVar(
		stringVar,
		"hls-output-dir",
		"/srv/flipcam/hls",
		"Sets the directory where the HLS segments and playlists are stored.",
	)
}

func addHlsUrlPathPrefixFlag(cmd *cobra.Command, stringVar *string) {
	cmd.Flags().StringVar(
		stringVar,
		"hls-url-path-prefix",
		"/camera",
		"Sets the path prefix for the playlist URL.",
	)
}

func addHostnameFlag(cmd *cobra.Command, stringVar *string) {
	cmd.Flags().StringVar(
		stringVar,
		"hostname",
		"",
		"If specified, flipcam will be hosted on this domain.",
	)
}

func addInterfaceFlag(cmd *cobra.Command, stringVar *string) {
	flagName := "wireless-interface"
	cmd.Flags().StringVar(
		stringVar,
		flagName,
		"",
		"Sets the name of the wireless interface to use.",
	)
	err := cmd.MarkFlagRequired(flagName)
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.RegisterFlagCompletionFunc(flagName, wifiInterfaceComplete)
	if err != nil {
		log.Fatalf("failed to register wireless interface completion: %v", err)
	}
}

func wifiInterfaceComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	matches := make([]string, 0)

	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Error getting network interfaces: %v\n", err)
		return nil, cobra.ShellCompDirectiveError
	}

	for _, iface := range interfaces {
		_, err := os.Stat(path.Join("/sys/class/net", iface.Name, "wireless"))
		if err != nil {
			continue
		}

		if len(toComplete) == 0 || !strings.HasPrefix(iface.Name, toComplete) {
			matches = append(matches, iface.Name)
		}
	}

	return matches, cobra.ShellCompDirectiveDefault
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

func addWpaPassphraseFlag(cmd *cobra.Command, v *wpaPassphraseFlag) {
	flagName := "wireless-passphrase"
	cmd.Flags().Var(
		v,
		flagName,
		"Sets the WPA passphrase for the AP. 8 to 63 printable ASCII characters.",
	)
	err := cmd.MarkFlagRequired(flagName)
	if err != nil {
		log.Fatal(err)
	}
}
