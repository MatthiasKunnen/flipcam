package flipcamlib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"slices"
	"strconv"
)

func (f *FlipCam) setupNetwork(ctx context.Context) {
	var shutdownFuncs []func()
	iface, err := net.InterfaceByName(f.wirelessInterface)
	if err != nil {
		f.stopWithError(fmt.Errorf("unable to find interface %s: %w", f.wirelessInterface, err))
		return
	}

	addrs, err := iface.Addrs()
	if err != nil {
		f.stopWithError(fmt.Errorf("unable to get interface addresses: %w", err))
		return
	}

	setIp := true

	for _, addr := range addrs {
		parsedPrefix, err := netip.ParsePrefix(addr.String())
		if err != nil {
			f.stopWithError(fmt.Errorf(
				"unable to parse address %s of interface %s: %w",
				addr.String(),
				f.wirelessInterface,
				err,
			))
			return
		}

		if f.RouterIp().Addr() == parsedPrefix.Addr() {
			setIp = false
		}
	}

	var nmSetManaged bool
	state, err := getNetworkManagerDeviceState(ctx, f.wirelessInterface)
	var exitError *exec.ExitError
	switch {
	case errors.Is(err, exec.ErrNotFound):
		// NetworkManager not used in system
	case errors.As(err, &exitError):
		switch exitError.ExitCode() {
		case NM_EXIT_NOT_RUNNING:
		default:
			f.stopWithError(fmt.Errorf(
				"nmcli exited with an unexpected exit code: %d, %v",
				exitError.ExitCode(),
				err,
			))
			return
		}
	case err != nil:
		f.stopWithError(fmt.Errorf("nmcli error: %v", err))
		return
	default:
		switch state {
		case NM_DEVICE_STATE_UNMANAGED:
		default:
			nmSetManaged = true
		}
	}

	if err != nil {
		f.stopWithError(err)
		return
	}

	go func() {
		defer f.shutdownWg.Done() // Add occurred in calling function
		<-f.stop

		for _, shutdownFunc := range slices.Backward(shutdownFuncs) {
			shutdownFunc()
		}
	}()

	if nmSetManaged {
		shutdownFuncs = append(shutdownFuncs, func() {
			// @todo add shutdown context
			err := sudoCommand(context.TODO(), f.nmcliEnableManagedCmd()).Run()
			if err != nil {
				f.addShutdownError(fmt.Errorf(
					"unable to enable management of interface: %w",
					err,
				))
			}
		})

		err := sudoCommand(ctx, f.nmcliDisableManagedCmd()).Run()
		if err != nil {
			f.stopWithError(fmt.Errorf("unable to disable management of interface: %w", err))
			return
		}
	}

	if setIp {
		shutdownFuncs = append(shutdownFuncs, func() {
			// @todo add shutdown context
			err := sudoCommand(context.TODO(), f.ipAddrRemove()).Run()
			if err != nil {
				f.addShutdownError(fmt.Errorf("unable to remove the IP of %s: %w", f.wirelessInterface, err))
			}
		})

		err := sudoCommand(context.TODO(), f.ipAddrAdd()).Run()
		if err != nil {
			f.stopWithError(fmt.Errorf(
				"unable to add IP %s to interface %s: %w",
				f.RouterIp().String(),
				f.wirelessInterface,
				err,
			))
			return
		}
	}

	f.startupWg.Done()
}

func (f *FlipCam) ipAddrAdd() []string {
	return []string{
		"/usr/bin/ip",
		"addr",
		"add",
		f.RouterIp().String(),
		"dev",
		f.wirelessInterface,
	}
}

func (f *FlipCam) ipAddrRemove() []string {
	return []string{
		"/usr/bin/ip",
		"addr",
		"delete",
		f.RouterIp().String(),
		"dev",
		f.wirelessInterface,
	}
}

func (f *FlipCam) nmcliDisableManagedCmd() []string {
	return []string{
		"/usr/bin/nmcli",
		"device",
		"set",
		f.wirelessInterface,
		"managed",
		"no",
	}
}

func (f *FlipCam) nmcliEnableManagedCmd() []string {
	return []string{
		"/usr/bin/nmcli",
		"device",
		"set",
		f.wirelessInterface,
		"managed",
		"yes",
	}
}

const (
	NM_EXIT_SUCCESS int = iota
	NM_EXIT_ERROR_UNKNOWN
	NM_EXIT_INVALID_INPUT
	NM_EXIT_TIMEOUT_EXPIRED
	NM_EXIT_CONNECTION_ACTIVATION_FAILED
	NM_EXIT_CONNECTION_DEACTIVATION_FAILED
	NM_EXIT_DISCONNECTING_DEVICE_FAILED
	NM_EXIT_CONNECTION_DELETION_FAILED
	NM_EXIT_NOT_RUNNING
	// NM_EXIT_NOT_FOUND indicates that the connection, device, or access point does not exist.
	NM_EXIT_NOT_FOUND = 10
)

// NmDeviceState represents the state of a network device in NetworkManager.
type NmDeviceState int

const (
	// NM_DEVICE_STATE_UNKNOWN indicates that the device's state is unknown.
	NM_DEVICE_STATE_UNKNOWN NmDeviceState = iota * 10

	// NM_DEVICE_STATE_UNMANAGED indicates that the device is recognized, but not managed by
	// NetworkManager.
	NM_DEVICE_STATE_UNMANAGED

	// NM_DEVICE_STATE_UNAVAILABLE indicates that the device is managed by NetworkManager, but
	// is not available for use.
	// Reasons may include the wireless switched off, missing firmware, no ethernet carrier,
	// missing supplicant or modem manager, etc.
	NM_DEVICE_STATE_UNAVAILABLE

	// NM_DEVICE_STATE_DISCONNECTED indicates that the device can be activated, but is currently
	// idle and not connected to a network.
	NM_DEVICE_STATE_DISCONNECTED

	// NM_DEVICE_STATE_PREPARE indicates that the device is preparing the connection to the network.
	// This may include operations like changing the MAC address, setting physical link properties,
	// and anything else required to connect to the requested network.
	NM_DEVICE_STATE_PREPARE

	// NM_DEVICE_STATE_CONFIG indicates that the device is connecting to the requested network.
	// This may include operations like associating with the WiFi AP, dialing the modem,
	// connecting to the remote Bluetooth device, etc.
	NM_DEVICE_STATE_CONFIG

	// NM_DEVICE_STATE_NEED_AUTH indicatest that the device requires more information to continue
	// connecting to the requested network.
	// This includes secrets like WiFi passphrases, login passwords, PIN codes, etc.
	NM_DEVICE_STATE_NEED_AUTH

	// NM_DEVICE_STATE_IP_CONFIG that the device is requesting IPv4 and/or IPv6 addresses and
	// routing information from the network.
	NM_DEVICE_STATE_IP_CONFIG

	// NM_DEVICE_STATE_IP_CHECK indicates that the device is checking whether further action is
	// required for the requested network connection.
	// This may include checking whether only local network access is available, whether a captive
	// portal is blocking access to the Internet, etc.
	NM_DEVICE_STATE_IP_CHECK

	// NM_DEVICE_STATE_SECONDARIES indicates that the device is waiting for a secondary connection
	// (like a VPN) which must activated before the device can be activated
	NM_DEVICE_STATE_SECONDARIES

	// NM_DEVICE_STATE_ACTIVATED indicates that the device has a network connection, either
	// local or global.
	NM_DEVICE_STATE_ACTIVATED

	// NM_DEVICE_STATE_DEACTIVATING indicates a disconnection from the current network connection
	// was requested, and the device is cleaning up resources used for that connection.
	// The network connection may still be valid.
	NM_DEVICE_STATE_DEACTIVATING

	// NM_DEVICE_STATE_FAILED indicates that the device failed to connect to the requested network
	// and is cleaning up the connection request.
	NM_DEVICE_STATE_FAILED
)

func getNetworkManagerDeviceState(ctx context.Context, device string) (NmDeviceState, error) {
	output, err := exec.CommandContext(
		ctx,
		"/usr/bin/nmcli",
		"--get-values",
		"general.state",
		"device",
		"show",
		device,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("unable to check if device %s is managed by NetworkManager: %w", device, err)
	}
	spaceIndex := bytes.IndexByte(output, ' ')
	if spaceIndex == -1 {
		return 0, fmt.Errorf("could not parse device state %s", output)
	}

	stateNumber, err := strconv.Atoi(string(output[:spaceIndex]))
	return NmDeviceState(stateNumber), nil
}
