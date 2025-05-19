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

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "Runs flipcam",
	Example: `flipcam run`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		stopSig := make(chan os.Signal, 1)
		signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)
		flipcam := flipcamlib.FlipCam{
			HlsOutputDir:     hlsOutputDir,
			HlsUrlPathPrefix: hlsUrlPathPrefix,
		}
		flipcam.Init()
		go func() {
			err := flipcam.Start()
			if err != nil {
				log.Fatalf("Failed to start flipcam: %v", err)
			}

			err = flipcam.Wait()
			if err != nil {
				log.Fatalf("Flipcam exited with error: %v", err)
			}
		}()
		select {
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
	runCmd.Flags().StringVar(
		&hlsOutputDir,
		"hls-output-dir",
		"",
		"Sets the directory where the HLS segments and playlists are stored.",
	)
	runCmd.Flags().StringVar(
		&hlsUrlPathPrefix,
		"hls-url-path-prefix",
		"/camera",
		"Sets the path prefix for the playlist URL.",
	)
	err := runCmd.MarkFlagRequired("hls-output-dir")
	if err != nil {
		log.Fatal(err)
	}
}
