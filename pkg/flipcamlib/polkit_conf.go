package flipcamlib

import (
	_ "embed"
	"io"
	"text/template"
)

//go:embed polkit.js
var polkitTmplString string
var polkitTmpl = template.Must(template.New("polkit.js").Parse(polkitTmplString))

func (f *FlipCam) GeneratePolkitRule(w io.Writer) error {
	return polkitTmpl.Execute(w, map[string]interface{}{
		"CaddyServiceName":   f.serviceNameCaddy,
		"DnsmasqServiceName": f.serviceNameDnsmasq,
		"HostapdServiceName": f.serviceNameHostapd,
	})
}
