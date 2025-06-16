package flipcamlib

import (
	_ "embed"
	"encoding/binary"
	"io"
	"net"
	"text/template"
)

type dnsmasqConf struct {
	DhcpEnd    string
	DhcpStart  string
	Interface  string
	RouterIp   string
	LocalHosts []string
}

var dnsmasqConfT = template.Must(template.New("dnsmasq_conf").Parse(`
{{- /* Trim leading spaces */ -}}
interface={{.Interface}}
dhcp-option=option:router,{{.RouterIp}}
dhcp-option=option:dns-server,{{.RouterIp}}
dhcp-range={{.DhcpStart}},{{.DhcpEnd}},24h

# Point these domains to the local IP
{{- range .LocalHosts}}
interface-name={{.}},{{$.Interface}}/4
{{- end}}
log-debug
`))

type DnsmasqConfOpts struct {
	Hostname string
}

func (f *FlipCam) GenerateDnsmasqConf(w io.Writer, opts DnsmasqConfOpts) error {
	routerIp := f.RouterIp()
	dhcpStart := routerIp.Addr().Next()

	networkAddr := routerIp.Masked().Addr().As4()
	networkAddrNumber := binary.BigEndian.Uint32(networkAddr[:])
	reverseMask := (uint32(1) << (32 - routerIp.Bits())) - 1
	broadcastAddress := networkAddrNumber | reverseMask

	dhcpEnd := make(net.IP, 4)
	binary.BigEndian.PutUint32(dhcpEnd, broadcastAddress-1)
	localHosts := []string{goProApi}
	if opts.Hostname != "" {
		localHosts = append(localHosts, opts.Hostname)
	}

	return dnsmasqConfT.Execute(w, dnsmasqConf{
		DhcpStart:  dhcpStart.String(),
		DhcpEnd:    dhcpEnd.String(),
		Interface:  f.WirelessInterface(),
		RouterIp:   routerIp.Addr().String(),
		LocalHosts: localHosts,
	})
}

//go:embed dnsmasq.service
var dnsmasqService string
var dnsmasqServiceTmpl = template.Must(template.New("dnsmasq.service").Parse(dnsmasqService))

type DnsmasqServiceOpts struct {
	ConfFilePath string
}

func (f *FlipCam) GenerateDnsmasqService(w io.Writer, opts DnsmasqServiceOpts) error {
	return dnsmasqServiceTmpl.Execute(w, opts)
}
