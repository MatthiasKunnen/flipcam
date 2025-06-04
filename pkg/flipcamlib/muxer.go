package flipcamlib

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"sync"
	"time"
)

func (f *FlipCam) runMuxer(ctx context.Context) {
	reportStarted := sync.OnceFunc(f.startupWg.Done)
	internalRestartChan := make(chan chan struct{}, 1)
	muxer := RtmpToHlsMuxer{
		Url: "rtmp://0.0.0.0:1935/camera/",
	}
	numOfRestarts := -1

	for {
		numOfRestarts++
		var prefix string
		for {
			prefix = rand.Text()[:6]
			_, err := os.Stat(path.Join(f.hlsOutputDir, prefix+".m3u8"))
			if errors.Is(err, os.ErrNotExist) {
				break
			}
		}
		muxer.Prefix = prefix + "_"
		newPlaylistFile := prefix + ".m3u8"
		muxer.PlaylistPath = path.Join(f.hlsOutputDir, newPlaylistFile)
		err := f.setPlayListUrlPath(newPlaylistFile)
		if err != nil {
			f.stopWithError(fmt.Errorf("[muxer]: failed to set playlist path: %w", err))
			return
		}

		err = muxer.Start()
		if err != nil {
			log.Printf("[muxer]: failed to start: %v\n", err)
			if numOfRestarts == 0 {
				// If the muxer fails to start on first start, give up
				f.stopWithError(fmt.Errorf("failed to start muxer: %w", err))
				return
			}
		}
		reportStarted()

		select {
		case done := <-internalRestartChan:
			close(done)
		default:
		}

		runEnd := make(chan struct{})
		if numOfRestarts > 0 {
			// The first Add occurred in calling function
			f.shutdownWg.Add(1)
		}
		go func() {
			defer f.shutdownWg.Done()
			select {
			case <-f.stop:
			case done := <-f.restartMuxer:
				if done != nil {
					internalRestartChan <- done
				}
			case <-runEnd:
				return
			}
			// @todo reuse Shutdown ctx
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
}

func (f *FlipCam) setPlayListUrlPath(playlistFile string) error {
	newPath, err := url.JoinPath(f.hlsUrlPathPrefix, playlistFile)
	if err != nil {
		return err
	}

	f.hlsPlayListPathMu.Lock()
	f.hlsPlayListPath = newPath
	f.hlsPlayListPathMu.Unlock()
	return nil
}
