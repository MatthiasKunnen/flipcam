package flipcamlib

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/MatthiasKunnen/systemctl"
	"github.com/MatthiasKunnen/systemctl/properties"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

var CaddyConfPath = "/etc/flipcam/caddy.json"
var DnsmasqConfPath = "/etc/flipcam/dnsmasq.conf"
var HostapdConfPath = "/etc/flipcam/hostapd.conf"

const DefaultServiceNameCaddy = "flipcam-caddy.service"
const DefaultServiceNameDnsmasq = "flipcam-dnsmasq.service"
const DefaultServiceNameHostapd = "flipcam-hostapd.service"

var origin = "https://flipcam.sd4u.be"
var host = "flipcam.sd4u.be"

type Opts struct {
	HlsOutputDir     string
	HlsUrlPathPrefix string

	RouterAddr netip.Prefix

	ServiceNameCaddy   string
	ServiceNameDnsmasq string
	ServiceNameHostapd string

	// The port on which the web UI will be bound. E.g. :3000.
	UiPort string

	WirelessInterface string
}

type FlipCam struct {
	hlsOutputDir string
	routerAddr   netip.Prefix

	serviceNameCaddy   string
	serviceNameDnsmasq string
	serviceNameHostapd string
	services           []string

	wirelessInterface string
	hlsPlayListPath   string
	hlsPlayListPathMu sync.RWMutex
	hlsUrlPathPrefix  string
	restartMuxer      chan chan struct{}
	shutdownErr       error
	shutdownErrMu     sync.Mutex
	shutdownOnce      sync.Once
	shutdownWg        sync.WaitGroup
	stop              chan struct{}
	stopped           chan struct{}
	uiPort            string
}

func New(opts Opts) *FlipCam {
	opts.ServiceNameCaddy = defaultString(opts.ServiceNameCaddy, DefaultServiceNameCaddy)
	opts.ServiceNameDnsmasq = defaultString(opts.ServiceNameDnsmasq, DefaultServiceNameDnsmasq)
	opts.ServiceNameHostapd = defaultString(opts.ServiceNameHostapd, DefaultServiceNameHostapd)

	f := &FlipCam{
		hlsOutputDir:     opts.HlsOutputDir,
		hlsUrlPathPrefix: opts.HlsUrlPathPrefix,

		restartMuxer: make(chan chan struct{}),
		routerAddr:   opts.RouterAddr,

		serviceNameCaddy:   opts.ServiceNameCaddy,
		serviceNameDnsmasq: opts.ServiceNameDnsmasq,
		serviceNameHostapd: opts.ServiceNameHostapd,
		services: []string{
			opts.ServiceNameCaddy,
			opts.ServiceNameDnsmasq,
			opts.ServiceNameHostapd,
		},

		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
		uiPort:  defaultString(opts.UiPort, ":3000"),

		wirelessInterface: opts.WirelessInterface,
	}

	return f
}

func (f *FlipCam) Start(ctx context.Context) error {
	var startupWg sync.WaitGroup

	for _, service := range f.services {
		err := systemctl.Start(ctx, service, systemctl.Options{})
		if err != nil {
			return fmt.Errorf("failed to start %s: %w", service, err)
		}
	}

	startupWg.Add(1)
	go func() {
		defer startupWg.Done()
		f.waitForServices(ctx)
	}()

	startupWg.Add(1)
	muxerStarted := sync.OnceFunc(startupWg.Done)
	f.shutdownWg.Add(1)
	go func() {
		defer f.shutdownWg.Done()
		f.runMuxer(ctx, muxerStarted)
	}()

	go func() {
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
			err := srv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				f.stopWithError(fmt.Errorf("[web]: http listener failed: %w", err))
			}
		}()
		f.shutdownWg.Add(1)
		go func() {
			defer f.shutdownWg.Done()
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
	}()

	go func() {
		f.shutdownWg.Wait()
		close(f.stopped)
	}()

	startupWg.Wait() // All services have started
	// @todo if something went wrong, stop here and return shutdown error

	f.shutdownWg.Add(1)
	go func() {
		defer f.shutdownWg.Done()
		<-f.stop
		for _, service := range f.services {
			err := systemctl.Stop(context.Background(), service, systemctl.Options{})
			if err != nil {
				f.addShutdownError(fmt.Errorf("failed to stop service %s: %w\n", service, err))
			}
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			starting, inactive, err := f.servicesStatus()
			if err != nil {
				f.stopWithError(fmt.Errorf("error while checking service status: %w", err))
				return
			}

			switch {
			case len(inactive) > 0:
				f.stopWithError(fmt.Errorf(
					"the following services are not active: %s",
					strings.Join(inactive, ", "),
				))
			case len(starting) > 0:
				f.stopWithError(fmt.Errorf(
					"the following services changed state from active to starting: %s",
					strings.Join(starting, ", "),
				))
			}
		}
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

func (f *FlipCam) RouterIp() netip.Prefix {
	return f.routerAddr
}

func (f *FlipCam) ServiceNameCaddy() string {
	return f.serviceNameCaddy
}

func (f *FlipCam) ServiceNameDnsmasq() string {
	return f.serviceNameDnsmasq
}

func (f *FlipCam) ServiceNameHostapd() string {
	return f.serviceNameHostapd
}

func (f *FlipCam) WirelessInterface() string {
	return f.wirelessInterface
}

func (f *FlipCam) addShutdownError(err error) {
	f.shutdownErrMu.Lock()
	defer f.shutdownErrMu.Unlock()
	f.shutdownErr = errors.Join(f.shutdownErr, err)
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

func (f *FlipCam) getPlayListUrlPath() string {
	f.hlsPlayListPathMu.RLock()
	defer f.hlsPlayListPathMu.RUnlock()
	return f.hlsPlayListPath
}

func (f *FlipCam) runMuxer(ctx context.Context, started func()) {
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
			log.Fatalf("Failed to set playlist path: %v", err)
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
		started()

		select {
		case done := <-internalRestartChan:
			close(done)
		default:
		}

		runEnd := make(chan struct{})
		f.shutdownWg.Add(1)
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

// servicesStatus returns the state of the services.
func (f *FlipCam) servicesStatus() (starting []string, inactive []string, err error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	starting = make([]string, 0)
	inactive = make([]string, 0)

	for _, service := range f.services {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			sState, sErr := systemctl.Show(ctx, service, properties.ActiveState, systemctl.Options{})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				err = sErr
				return
			}

			switch sState {
			case "active":
			case "inactive", "failed", "deactivating", "maintenance":
				inactive = append(inactive, service)
			case "activating", "reloading", "refreshing":
				starting = append(starting, service)
			}
		}()
	}

	wg.Wait()
	return
}

func (f *FlipCam) stopWithError(err error) {
	f.addShutdownError(err)
	_ = f.Shutdown(context.Background())
}

// waitForServices returns when it can be determined if the services have started successfully.
func (f *FlipCam) waitForServices(ctx context.Context) {
	for {
		starting, inactive, err := f.servicesStatus()
		if err != nil {
			f.stopWithError(fmt.Errorf("error while waiting for services to become active: %w", err))
			return
		}

		switch {
		case len(inactive) > 0:
			f.stopWithError(fmt.Errorf(
				"the following services failed to become active: %s",
				strings.Join(inactive, ", "),
			))
			return
		case len(starting) > 0:
			select {
			case <-ctx.Done():
				f.stopWithError(fmt.Errorf(
					"the following services failed to become active in time: %s",
					strings.Join(inactive, ", "),
				))
			case <-time.After(1 * time.Second):
				continue
			}
		}

		return
	}
}
