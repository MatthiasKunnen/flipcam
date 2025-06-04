package flipcamlib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func (f *FlipCam) startWebserver(ctx context.Context) {
	srv := http.Server{Addr: f.uiPort}
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
		f.restartMuxer <- done
		<-done
		w.Header().Set("Content-Type", "text/plain")
		_, err := w.Write([]byte(f.getPlayListUrlPath()))
		if err != nil {
			log.Printf("web: failed to write new playlist path: %v\n", err)
		}
	})

	f.shutdownWg.Add(1)
	go func() {
		defer f.shutdownWg.Done()
		log.Println("Listening on :3000")
		f.startupWg.Done()
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			f.stopWithError(fmt.Errorf("[web]: http listener failed: %w", err))
		}
	}()

	go func() {
		defer f.shutdownWg.Done() // Add occurred in calling function
		<-f.stop
		// @todo We should use the shutdown context here
		ctx, release := context.WithTimeout(context.Background(), 5*time.Second)
		defer release()
		err := srv.Shutdown(ctx)
		if err != nil {
			f.addShutdownError(fmt.Errorf("[web]: error during shutdown: %w", err))
		} else {
			log.Println("[web]: shutdown cleanly")
		}
	}()
}
