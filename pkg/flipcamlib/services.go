package flipcamlib

import (
	"context"
	"fmt"
	"github.com/MatthiasKunnen/systemctl"
	"github.com/MatthiasKunnen/systemctl/properties"
	"strings"
	"sync"
	"time"
)

func (f *FlipCam) startServices(ctx context.Context) {
	go func() {
		// Cleanup
		defer f.shutdownWg.Done() // Add occurred in calling function
		<-f.stop
		for _, service := range f.services {
			err := systemctl.Stop(context.Background(), service, systemctl.Options{})
			if err != nil {
				f.addShutdownError(fmt.Errorf("failed to stop service %s: %w\n", service, err))
			}
		}
	}()

	for _, service := range f.services {
		err := systemctl.Start(ctx, service, systemctl.Options{})
		if err != nil {
			f.stopWithError(fmt.Errorf("failed to start service %s: %w", service, err))
			return
		}
	}

	go func() {
		for {
			starting, inactive, err := servicesStatus(f.services)
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
				case <-f.stop:
					return
				case <-ctx.Done():
					f.stopWithError(fmt.Errorf(
						"the following services failed to become active in time: %s",
						strings.Join(inactive, ", "),
					))
				case <-time.After(1 * time.Second):
					continue
				}
			}

			f.startupWg.Done()
			return
		}
	}()

	f.shutdownWg.Add(1)
	go func() {
		// Monitor services when started
		defer f.shutdownWg.Done()

		select {
		case <-f.started:
		case <-f.stop:
			return
		}

		for {
			select {
			case <-f.stop:
				return
			case <-time.After(10 * time.Second):
			}
			starting, inactive, err := servicesStatus(f.services)
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
}

// servicesStatus returns the state of the services.
func servicesStatus(services []string) (starting []string, inactive []string, err error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	starting = make([]string, 0)
	inactive = make([]string, 0)

	for _, service := range services {
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
