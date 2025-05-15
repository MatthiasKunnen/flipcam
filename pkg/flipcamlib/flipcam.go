package flipcamlib

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type FlipCam struct {
	HlsOutputDir string
	stop         chan struct{}
	stopped      chan struct{}
}

func (f *FlipCam) Init() {
	f.stop = make(chan struct{})
	f.stopped = make(chan struct{})
}

func (f *FlipCam) Start() error {
	var wg sync.WaitGroup
	muxer := RtmpToHlsMuxer{
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
				_, err := os.Stat(path.Join(f.HlsOutputDir, prefix+".m3u8"))
				if errors.Is(err, os.ErrNotExist) {
					break
				}
			}
			muxer.Prefix = prefix + "_"
			muxer.PlaylistPath = path.Join(f.HlsOutputDir, prefix+".m3u8")
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
				case <-f.stop:
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
