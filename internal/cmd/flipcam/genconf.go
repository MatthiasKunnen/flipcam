package flipcam

import (
	_ "embed"
	"errors"
	"github.com/MatthiasKunnen/flipcam/pkg/flipcamlib"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
	"text/template"
)

//go:embed install.sh
var installScript string
var installTmpl = template.Must(template.New("install.sh").Parse(installScript))
var caddyBinaryPath string

var genConfCmd = &cobra.Command{
	Use:     "genconf",
	Short:   "Generates the configuration files for flipcam",
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

		outDir := "conf_out"
		err := os.Mkdir(outDir, 0755)
		switch {
		case errors.Is(err, os.ErrExist):
		case err != nil:
			log.Fatalf("failed to create %s directory: %v", outDir, err)
		}

		caddyFilename := "caddy.json"
		caddyF, err := os.Create(path.Join(outDir, caddyFilename))
		if err != nil {
			log.Fatalf("failed to create caddy JSON file: %v", err)
		}
		err = flipcam.GenerateCaddyConfig(caddyF)
		if err != nil {
			log.Fatalf("failed to generate caddy config: %v", err)
		}
		err = caddyF.Close()
		if err != nil {
			log.Fatalf("failed to close caddy config file: %v", err)
		}

		caddyServiceF, err := os.Create(path.Join(outDir, flipcam.ServiceNameCaddy()))
		if err != nil {
			log.Fatalf("failed to create %s: %v", flipcam.ServiceNameCaddy(), err)
		}
		err = flipcam.GenerateCaddyServiceUnit(caddyServiceF, flipcamlib.CaddyServiceUnitOptions{
			BinaryPath: caddyBinaryPath,
			ConfigPath: "/etc/flipcam/caddy.json",
		})
		if err != nil {
			log.Fatalf("failed to generate %s: %v", flipcam.ServiceNameCaddy(), err)
		}
		err = caddyServiceF.Close()
		if err != nil {
			log.Fatalf("failed to close %s: %v", flipcam.ServiceNameCaddy(), err)
		}

		dnsmasqFilename := "dnsmasq.conf"
		dnsmasqF, err := os.Create(path.Join(outDir, dnsmasqFilename))
		if err != nil {
			log.Fatalf("failed to create dnsmasq config file: %v", err)
		}
		err = flipcam.GenerateDnsmasqConf(dnsmasqF)
		if err != nil {
			log.Fatalf("failed to generate dnsmasq config: %v", err)
		}
		err = dnsmasqF.Close()
		if err != nil {
			log.Fatalf("failed to close dnsmasq config file: %v", err)
		}

		dnsmasqServiceF, err := os.Create(path.Join(outDir, flipcam.ServiceNameDnsmasq()))
		if err != nil {
			log.Fatalf("failed to create dnsmasq service: %v", err)
		}
		err = flipcam.GenerateDnsmasqService(dnsmasqServiceF, flipcamlib.DnsmasqServiceOpts{
			ConfFilePath: flipcamlib.DnsmasqConfPath,
		})
		if err != nil {
			log.Fatalf("failed to generate dnsmasq service: %v", err)
		}
		err = dnsmasqServiceF.Close()
		if err != nil {
			log.Fatalf("failed to close dnsmasq service file: %v", err)
		}

		hostapdFilename := "hostapd.conf"
		hostapdF, err := os.Create(path.Join(outDir, hostapdFilename))
		if err != nil {
			log.Fatalf("failed to create hostapd config file: %v", err)
		}
		err = flipcam.GenerateHostapdConf(hostapdF)
		if err != nil {
			log.Fatalf("failed to generate hostapd config: %v", err)
		}
		err = hostapdF.Close()
		if err != nil {
			log.Fatalf("failed to close hostapd config file: %v", err)
		}

		hostapdServiceF, err := os.Create(path.Join(outDir, flipcam.ServiceNameHostapd()))
		if err != nil {
			log.Fatalf("failed to create hostapd service: %v", err)
		}
		err = flipcam.GenerateHostapdService(hostapdServiceF, flipcamlib.HostapdServiceOpts{
			ConfFilePath: flipcamlib.HostapdConfPath,
		})
		if err != nil {
			log.Fatalf("failed to generate hostapd service: %v", err)
		}
		err = hostapdServiceF.Close()
		if err != nil {
			log.Fatalf("failed to close hostapd service file: %v", err)
		}

		polkitFilename := "polkit.js"
		polkitF, err := os.Create(path.Join(outDir, polkitFilename))
		if err != nil {
			log.Fatalf("failed to create polkit.js: %v", err)
		}
		err = flipcam.GeneratePolkitRule(polkitF)
		if err != nil {
			log.Fatalf("failed to generate polkit.js: %v", err)
		}
		err = polkitF.Close()
		if err != nil {
			log.Fatalf("failed to close polkit.js file: %v", err)
		}

		sudoersFilename := "sudoers.conf"
		sudoersF, err := os.Create(path.Join(outDir, sudoersFilename))
		if err != nil {
			log.Fatalf("failed to create sudoers config file: %v", err)
		}
		err = flipcam.GenerateSudoersConf(sudoersF)
		if err != nil {
			log.Fatalf("failed to generate sudoers config: %v", err)
		}
		err = sudoersF.Close()
		if err != nil {
			log.Fatalf("failed to close sudoers config file: %v", err)
		}

		installF, err := os.OpenFile(
			path.Join(outDir, "install.sh"),
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			0777,
		)
		if err != nil {
			return
		}
		err = installTmpl.Execute(installF, map[string]interface{}{
			"CaddyConfPath":      flipcamlib.CaddyConfPath,
			"CaddyServiceName":   flipcam.ServiceNameCaddy(),
			"DnsmasqConfPath":    flipcamlib.DnsmasqConfPath,
			"DnsmasqServiceName": flipcam.ServiceNameDnsmasq(),
			"HostapdConfPath":    flipcamlib.HostapdConfPath,
			"HostapdServiceName": flipcam.ServiceNameHostapd(),
			"PolkitFilename":     polkitFilename,
		})
		if err != nil {
			log.Fatalf("failed to generate install script: %v", err)
		}
		err = installF.Close()
		if err != nil {
			log.Fatalf("failed to close install script file: %v", err)
		}
	},
}

func init() {
	addHlsOutputDirFlag(genConfCmd, &hlsOutputDir)
	addHlsUrlPathPrefixFlag(genConfCmd, &hlsUrlPathPrefix)
	addInterfaceFlag(genConfCmd, &wirelessInterface)
	addIpv4Flag(genConfCmd, &routerIp)
	addUiPortFlag(genConfCmd, &hlsUrlPathPrefix)
	genConfCmd.Flags().StringVar(
		&caddyBinaryPath,
		"caddy-binary-path",
		"/usr/bin/caddy",
		"Sets the path to the caddy binary.",
	)
}
