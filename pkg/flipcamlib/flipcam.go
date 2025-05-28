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

var origin = "https://flipcam.sd4u.be"
var host = "flipcam.sd4u.be"

type FlipCam struct {
	HlsOutputDir      string
	HlsUrlPathPrefix  string
	UiPort            string
	hlsPlayListPath   string
	hlsPlayListPathMu sync.RWMutex
	shutdownErr       error
	shutdownErrMu     sync.Mutex
	shutdownOnce      sync.Once
	stop              chan struct{}
	stopped           chan struct{}
}

func (f *FlipCam) Init() {
	f.stop = make(chan struct{})
	f.stopped = make(chan struct{})
	if f.UiPort == "" {
		f.UiPort = ":3000"
	}
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

			err = muxer.Start()
			if err != nil {
				log.Printf("[muxer]: failed to start: %v\n", err)
			}

			select {
			case done := <-internalRestartChan:
				close(done)
			default:
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
					f.addShutdownError(fmt.Errorf("[muxer]: error during shutdown: %w", err))
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
		srv := http.Server{Addr: f.UiPort}
		static := http.FileServer(http.Dir("./static"))
		http.Handle("/static/", http.StripPrefix("/static/", static))

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			playlistUrlPath := origin + f.getPlayListUrlPath()
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
				_ = f.Shutdown(context.Background())
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
				f.addShutdownError(fmt.Errorf("[web]: error during shutdown: %w", err))
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
	f.shutdownErrMu.Lock()
	defer f.shutdownErrMu.Unlock()
	return f.shutdownErr
}

// Shutdown stops flipcam and makes Wait return.
// If any error occurs, both Shutdown and Wait will return it.
// Shutdown is goroutine safe and can be called multiple times.
func (f *FlipCam) Shutdown(ctx context.Context) error {
	f.shutdownOnce.Do(func() {
		close(f.stop)
		select {
		case <-f.stopped:
		case <-ctx.Done():
			f.addShutdownError(ctx.Err())
		}
	})

	f.shutdownErrMu.Lock()
	defer f.shutdownErrMu.Unlock()
	return f.shutdownErr
}

func (f *FlipCam) addShutdownError(err error) {
	f.shutdownErrMu.Lock()
	defer f.shutdownErrMu.Unlock()
	f.shutdownErr = errors.Join(f.shutdownErr, err)
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
