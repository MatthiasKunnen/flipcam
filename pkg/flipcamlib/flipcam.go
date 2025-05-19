package flipcamlib

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"
)

type FlipCam struct {
	HlsOutputDir      string
	HlsUrlPathPrefix  string
	hlsPlayListPath   string
	hlsPlayListPathMu sync.RWMutex
	stop              chan struct{}
	stopped           chan struct{}
}

func (f *FlipCam) Init() {
	f.stop = make(chan struct{})
	f.stopped = make(chan struct{})
}

func (f *FlipCam) Start() error {
	var wg sync.WaitGroup
	restartMuxer := make(chan chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		internalRestartChan := make(chan chan struct{}, 1)
		muxer := RtmpToHlsMuxer{
			Url: "rtmp://0.0.0.0:1935/camera/",
		}

		for {
			var prefix string
			for {
				prefix = rand.Text()[:6]
				_, err := os.Stat(path.Join(f.HlsOutputDir, prefix+".m3u8"))
				if errors.Is(err, os.ErrNotExist) {
					break
				}
			}
			muxer.Prefix = prefix + "_"
			newPlaylistFile := prefix + ".m3u8"
			muxer.PlaylistPath = path.Join(f.HlsOutputDir, newPlaylistFile)
			err := f.setPlayListUrlPath(newPlaylistFile)
			if err != nil {
				log.Fatalf("Failed to set playlist path: %v", err)
			}

			select {
			case done := <-internalRestartChan:
				close(done)
			default:
			}

			err = muxer.Start()
			if err != nil {
				log.Printf("[muxer]: failed to start: %v\n", err)
			}

			runEnd := make(chan struct{})
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case <-f.stop:
				case done := <-restartMuxer:
					if done != nil {
						internalRestartChan <- done
					}
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
			case <-f.stop:
				// If the muxer was stopped by request from the main thread, do not restart
				log.Println("[muxer]: shutdown cleanly")
				return
			default:
				log.Println("[muxer]: restarting")
				time.Sleep(1 * time.Second)
			}
		}
	}()

	go func() {
		srv := http.Server{Addr: ":3000"}
		static := http.FileServer(http.Dir("./static"))
		http.Handle("/static/", http.StripPrefix("/static/", static))

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			playlistUrlPath := "https://flipcam.sd4u.be" + f.getPlayListUrlPath()
			err := Index(playlistUrlPath).Render(r.Context(), w)
			if err != nil {
				log.Printf("web: failed to render: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})

		http.HandleFunc("/restart-muxer", func(w http.ResponseWriter, r *http.Request) {
			done := make(chan struct{})
			restartMuxer <- done
			<-done
			w.Header().Set("Content-Type", "text/plain")
			_, err := w.Write([]byte(f.getPlayListUrlPath()))
			if err != nil {
				log.Printf("web: failed to write new playlist path: %v\n", err)
			}
		})

		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Listening on :3000")
			err := srv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("web: http listener failed: %v\n", err)
				f.Shutdown(context.Background())
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-f.stop
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

	go func() {
		wg.Wait()
		close(f.stopped)
	}()
	return nil
}

func (f *FlipCam) Wait() error {
	<-f.stopped
	return nil
}

func (f *FlipCam) Shutdown(ctx context.Context) error {
	close(f.stop)
	select {
	case <-f.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *FlipCam) setPlayListUrlPath(playlistFile string) error {
	newPath, err := url.JoinPath(f.HlsUrlPathPrefix, playlistFile)
	if err != nil {
		return err
	}

	f.hlsPlayListPathMu.Lock()
	f.hlsPlayListPath = newPath
	f.hlsPlayListPathMu.Unlock()
	return nil
}

func (f *FlipCam) getPlayListUrlPath() string {
	f.hlsPlayListPathMu.RLock()
	defer f.hlsPlayListPathMu.RUnlock()
	return f.hlsPlayListPath
}
