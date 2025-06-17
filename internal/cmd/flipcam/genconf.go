package flipcam

import (
	_ "embed"
	"github.com/MatthiasKunnen/flipcam/pkg/flipcamlib"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
)

var caddyBinaryPath string
var hostname string
var wpaPassphrase wpaPassphraseFlag

var genConfCmd = &cobra.Command{
	Use:     "genconf",
	Short:   "Generates configuration files for flipcam",
	Long:    `Generates ansible/generated_vars.yaml and ansible/files/caddy.json`,
	Example: `flipcam genconf --hls-output-dir /tmp/hls`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		flipcam := flipcamlib.New(flipcamlib.Opts{
			HlsOutputDir:      hlsOutputDir,
			HlsUrlPathPrefix:  hlsUrlPathPrefix,
			RouterAddr:        routerIp.Prefix(),
			UiPort:            uiPort,
			WirelessInterface: wirelessInterface,
		})

		outDir := "ansible"

		caddyFilename := "caddy.json"
		caddyF, err := os.Create(path.Join(outDir, "templates", caddyFilename))
		if err != nil {
			log.Fatalf("failed to create caddy JSON file: %v", err)
		}
		err = flipcam.GenerateCaddyConfig(caddyF, flipcamlib.CaddyConfOpts{
			Hostname: hostname,
		})
		if err != nil {
			log.Fatalf("failed to generate caddy config: %v", err)
		}
		err = caddyF.Close()
		if err != nil {
			log.Fatalf("failed to close caddy config file: %v", err)
		}

		err = rootCmd.GenBashCompletionFileV2(path.Join(outDir, "files/completion_bash"), true)
		if err != nil {
			log.Fatalf("failed to generate bash completion file: %v", err)
		}

		err = rootCmd.GenFishCompletionFile(path.Join(outDir, "files/completion_fish"), true)
		if err != nil {
			log.Fatalf("failed to generate fish completion file: %v", err)
		}

		err = rootCmd.GenZshCompletionFile(path.Join(outDir, "files/completion_zsh"))
		if err != nil {
			log.Fatalf("failed to generate zsh completion file: %v", err)
		}

		generatedVarsFilename := "generated_vars.yaml"
		generatedVarsF, err := os.Create(path.Join(outDir, generatedVarsFilename))
		if err != nil {
			log.Fatalf("failed to open %s: %v", generatedVarsFilename, err)
		}
		err = flipcam.GenerateVars(generatedVarsF, flipcamlib.GenerateVarsOpts{
			Hostname:           hostname,
			CaddyBinaryPath:    caddyBinaryPath,
			WirelessPassphrase: wpaPassphrase.String(),
		})
		if err != nil {
			log.Fatalf("failed to generate vars: %v", err)
		}
		err = generatedVarsF.Close()
		if err != nil {
			log.Fatalf("failed to close %s file: %v", generatedVarsFilename, err)
		}
	},
}

func init() {
	addHlsOutputDirFlag(genConfCmd, &hlsOutputDir)
	addHlsUrlPathPrefixFlag(genConfCmd, &hlsUrlPathPrefix)
	addHostnameFlag(genConfCmd, &hostname)
	addInterfaceFlag(genConfCmd, &wirelessInterface)
	addIpv4Flag(genConfCmd, &routerIp)
	addUiPortFlag(genConfCmd, &hlsUrlPathPrefix)
	addWpaPassphraseFlag(genConfCmd, &wpaPassphrase)
	genConfCmd.Flags().StringVar(
		&caddyBinaryPath,
		"caddy-binary-path",
		"/usr/bin/caddy",
		"Sets the path to the caddy binary.",
	)
}
