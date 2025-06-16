package flipcamlib

import (
	"context"
	"errors"
	"github.com/MatthiasKunnen/flipcam/pkg/chanwg"
	"net/netip"
	"sync"
)

var CaddyConfPath = "/etc/flipcam/caddy.json"
var DnsmasqConfPath = "/etc/flipcam/dnsmasq.conf"
var HostapdConfPath = "/etc/flipcam/hostapd.conf"

const DefaultServiceNameCaddy = "flipcam-caddy.service"
const DefaultServiceNameDnsmasq = "flipcam-dnsmasq.service"
const DefaultServiceNameHostapd = "flipcam-hostapd.service"

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

	// WaitGroup that finishes when all parts are shut down.
	shutdownWg chanwg.WaitGroup

	// Channel closed when every part is running.
	started <-chan struct{}

	// WaitGroup that finishes when every part is started.
	startupWg chanwg.WaitGroup

	// Channel closed when shutdown is requested.
	stop chan struct{}

	// Channel closed when everything has shut down.
	stopped <-chan struct{}

	uiPort string
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

		stop:   make(chan struct{}),
		uiPort: defaultString(opts.UiPort, ":3000"),

		wirelessInterface: opts.WirelessInterface,
	}
	f.started = f.startupWg.WaitChan()
	f.stopped = f.shutdownWg.WaitChan()

	return f
}

func (f *FlipCam) Start(ctx context.Context) error {
	startFuncs := []func(ctx context.Context){
		f.setupNetwork,
		f.runMuxer,
		f.startWebserver,
	}
	// functions must call startupWg.Done() if they started successfully.
	// Additional startupWg.Add/Done calls inside the function are allowed.
	// Where possible, they should use the context to allow the startup to be canceled.
	// If an irrecoverable error occurred, FlipCam.stopWithError must be called.
	f.startupWg.Add(len(startFuncs))
	// functions must shut down after receiving from <-FlipCam.stop.
	// Additional shutdownWg.Add/Done calls inside the function are allowed.
	// They must call shutdownWg.Done() after shutting down their respective resources, regardless
	// of success. If they don't have anything to shutdown, shutdownWg.Done() must still be called.
	// If any errors occur during shutdown, they must report their errors with
	// FlipCam.addShutdownError.
	f.shutdownWg.Add(len(startFuncs))
	for _, start := range startFuncs {
		go start(ctx)
	}

	select {
	case <-f.stop:
		f.shutdownErrMu.Lock()
		defer f.shutdownErrMu.Unlock()
		return f.shutdownErr
	case <-f.started:
		return nil
	}
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

func (f *FlipCam) getPlayListUrlPath() string {
	f.hlsPlayListPathMu.RLock()
	defer f.hlsPlayListPathMu.RUnlock()
	return f.hlsPlayListPath
}

func (f *FlipCam) stopWithError(err error) {
	f.addShutdownError(err)
	go func() {
		_ = f.Shutdown(context.Background())
	}()
}
