package flipcam

import (
	"github.com/MatthiasKunnen/flipcam/pkg/flipcamlib"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var genConfCmd = &cobra.Command{
	Use:     "genconf",
	Short:   "Generates the configuration files for flipcam",
	Example: `flipcam genconf --hls-output-dir /tmp/hls`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		flipcam := flipcamlib.New(flipcamlib.Opts{
			HlsOutputDir:     hlsOutputDir,
			HlsUrlPathPrefix: hlsUrlPathPrefix,
			UiPort:           uiPort,
		})
		f, err := os.Create("caddyFromGo.json")
		if err != nil {
			log.Fatal(err)
		}
		caddyConf := flipcam.GenerateCaddyConfig()
		err = json.MarshalWrite(f, caddyConf, jsontext.WithIndent("\t"))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	addHlsOutputDirFlag(genConfCmd, &hlsOutputDir)
	addHlsUrlPathPrefixFlag(genConfCmd, &hlsUrlPathPrefix)
	addUiPortFlag(genConfCmd, &hlsUrlPathPrefix)
}
