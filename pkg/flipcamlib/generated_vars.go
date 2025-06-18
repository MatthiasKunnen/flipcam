package flipcamlib

import (
	_ "embed"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"text/template"
)

type generatedVarsData struct {
	CaddyBinaryPath     string
	CaddyServiceName    string
	ConnectivityHosts   []string
	DhcpEnd             string
	DhcpStart           string
	DnsmasqServiceName  string
	FlipcamSudoCommands []string
	HlsOutputDir        string
	HostapdServiceName  string
	RouterIp            string
	WebDomain           string
	WirelessInterface   string
	WirelessPassphrase  string
}

type GenerateVarsOpts struct {
	CaddyBinaryPath    string
	Hostname           string
	WirelessPassphrase string
}

//go:embed generated_vars.yaml.tmpl
var generatedVarsTemplateString string
var generatedVarsTmpl = template.Must(template.New("generated_vars.yaml").
	Parse(generatedVarsTemplateString))

func (f *FlipCam) GenerateVars(w io.Writer, opts GenerateVarsOpts) error {
	routerIp := f.RouterIp()
	dhcpStart := routerIp.Addr().Next()

	networkAddr := routerIp.Masked().Addr().As4()
	networkAddrNumber := binary.BigEndian.Uint32(networkAddr[:])
	reverseMask := (uint32(1) << (32 - routerIp.Bits())) - 1
	broadcastAddress := networkAddrNumber | reverseMask

	dhcpEnd := make(net.IP, 4)
	binary.BigEndian.PutUint32(dhcpEnd, broadcastAddress-1)

	return generatedVarsTmpl.Execute(w, generatedVarsData{
		CaddyBinaryPath:    opts.CaddyBinaryPath,
		CaddyServiceName:   f.serviceNameCaddy,
		ConnectivityHosts:  []string{goProApi},
		DhcpEnd:            dhcpEnd.String(),
		DhcpStart:          dhcpStart.String(),
		DnsmasqServiceName: f.serviceNameDnsmasq,
		FlipcamSudoCommands: []string{
			strings.Join(f.ipAddrAdd(), " "),
			strings.Join(f.ipAddrRemove(), " "),
			strings.Join(f.nmcliDisableManagedCmd(), " "),
			strings.Join(f.nmcliEnableManagedCmd(), " "),
		},
		HlsOutputDir:       f.hlsOutputDir,
		HostapdServiceName: f.serviceNameHostapd,
		RouterIp:           routerIp.Addr().String(),
		WebDomain:          opts.Hostname,
		WirelessInterface:  f.wirelessInterface,
		WirelessPassphrase: opts.WirelessPassphrase,
	})
}
