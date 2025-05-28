package flipcam

import (
	"context"
	"github.com/MatthiasKunnen/flipcam/pkg/flipcamlib"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var hlsOutputDir string
var hlsUrlPathPrefix string
var uiPort string

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "Runs flipcam",
	Example: `flipcam run`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		stopSig := make(chan os.Signal, 1)
		signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)
		flipcam := flipcamlib.New(flipcamlib.Opts{
			HlsOutputDir:     hlsOutputDir,
			HlsUrlPathPrefix: hlsUrlPathPrefix,
			UiPort:           uiPort,
		})
		flipcamStopped := make(chan error)
		go func() {
			err := flipcam.Start()
			if err != nil {
				log.Fatalf("Failed to start flipcam: %v", err)
			}

			flipcamStopped <- flipcam.Wait()
		}()
		select {
		case err := <-flipcamStopped:
			if err != nil {
				log.Fatalf("Flipcam exited with error: %v", err)
			}
		case <-stopSig:
			log.Println("Exit signal received, stopping flipcam.")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*6)
			defer cancel()
			err := flipcam.Shutdown(ctx)
			if err != nil {
				log.Fatalf("Shutdown err: %v", err)
			}
		}
	},
}

func init() {
	addHlsOutputDirFlag(runCmd, &hlsOutputDir)
	addHlsUrlPathPrefixFlag(runCmd, &hlsUrlPathPrefix)
	addUiPortFlag(runCmd, &uiPort)
}
