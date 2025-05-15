package flipcam

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/MatthiasKunnen/flipcam/pkg/flipcamlib"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"
)

var hlsOutputDir string

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "Runs flipcam",
	Example: `flipcam run`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		stopSig := make(chan os.Signal, 1)
		signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

		var wg sync.WaitGroup
		stop := make(chan struct{})
		muxer := flipcamlib.RtmpToHlsMuxer{
			Url: "rtmp://0.0.0.0:1935/camera/",
		}
		var playlistPath string
		playlistPathMutex := sync.RWMutex{}
		restartMuxer := make(chan struct{})
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				var prefix string
				for {
					prefix = rand.Text()[:6]
					_, err := os.Stat(path.Join(hlsOutputDir, prefix+".m3u8"))
					if errors.Is(err, os.ErrNotExist) {
						break
					}
				}
				muxer.Prefix = prefix + "_"
				muxer.PlaylistPath = path.Join(hlsOutputDir, prefix+".m3u8")
				playlistPathMutex.Lock()
				playlistPath = muxer.PlaylistPath
				playlistPathMutex.Unlock()

				err := muxer.Start()
				if err != nil {
					log.Printf("[muxer]: failed to start: %v\n", err)
				}

				runEnd := make(chan struct{})
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case <-stop:
					case <-restartMuxer:
					case <-runEnd:
						return
					}
					ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
					defer cancel()
					err := muxer.Shutdown(ctx)
					if err != nil {
						log.Printf("[muxer]: failed to shutdown: %v\n", err)
					}
				}()

				err = muxer.Wait()
				if err != nil && !errors.Is(err, os.ErrProcessDone) {
					log.Printf("[muxer]: exited with error: %v\n", err)
				}
				close(runEnd)

				select {
				case <-stop:
					// If the muxer was stopped by request from the main thread, do not restart
					log.Println("[muxer]: shutdown cleanly")
					return
				default:
					log.Println("[muxer]: restarting")
					time.Sleep(1 * time.Second)
				}
			}
		}()

		func() {
			srv := http.Server{Addr: ":3000"}
			static := http.FileServer(http.Dir("./static"))
			http.Handle("/static/", http.StripPrefix("/static/", static))

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				playlistPathMutex.RLock()
				playlistUrlPath := "https://flipcam.sd4u.be/camera/" + path.Base(playlistPath)
				playlistPathMutex.RUnlock()
				err := Index(playlistUrlPath).Render(r.Context(), w)
				if err != nil {
					log.Printf("web: failed to render: %v\n", err)
					w.WriteHeader(http.StatusInternalServerError)
				}
			})

			wg.Add(1)
			go func() {
				defer wg.Done()
				fmt.Println("Listening on :3000")
				err := srv.ListenAndServe()
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Printf("web: http listener failed: %v\n", err)
					stopSig <- nil // We don't recover from this, let's stop everything
				}
			}()
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-stop
				ctx, release := context.WithTimeout(context.Background(), 5*time.Second)
				defer release()
				err := srv.Shutdown(ctx)
				if err != nil {
					log.Printf("web: error during shutdown: %v\n", err)
				} else {
					log.Println("web: shutdown cleanly")
				}
			}()
		}()

		select {
		case <-stopSig:
			log.Printf("stopping...")
			close(stop)
		}

		allStoppedChan := make(chan struct{})
		go func() {
			wg.Wait()
			close(allStoppedChan)
		}()
		select {
		case <-allStoppedChan:
		case <-time.After(6 * time.Second):
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
	err := runCmd.MarkFlagRequired("hls-output-dir")
	if err != nil {
		log.Fatal(err)
	}
}
