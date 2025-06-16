package flipcamlib

import (
	_ "embed"
	"io"
	"text/template"
)

//go:embed hostapd.conf
var hostapdConfTemplString string
var hostapdConfTmpl = template.Must(template.New("hostapd.conf").Parse(hostapdConfTemplString))

type HostapdConfOpts struct {
	WpaPassphrase string
}

func (f *FlipCam) GenerateHostapdConf(w io.Writer, opts HostapdConfOpts) error {
	return hostapdConfTmpl.Execute(w, map[string]interface{}{
		"Interface":     f.wirelessInterface,
		"WpaPassphrase": opts.WpaPassphrase,
	})
}

type HostapdServiceOpts struct {
	ConfFilePath string
}

//go:embed hostapd.service
var hostapdServiceString string
var hostapdServiceTmpl = template.Must(template.New("hostapd.service").Parse(hostapdServiceString))

func (f *FlipCam) GenerateHostapdService(w io.Writer, opts HostapdServiceOpts) error {
	return hostapdServiceTmpl.Execute(w, opts)
}
