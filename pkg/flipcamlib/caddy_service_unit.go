package flipcamlib

import (
	"io"
	"text/template"
)
import _ "embed"

//go:embed caddy.service
var caddyService string
var caddyServiceTmpl = template.Must(template.New("caddy.service").Parse(caddyService))

type CaddyServiceUnitOptions struct {
	BinaryPath string
	ConfigPath string
}

func (f *FlipCam) GenerateCaddyServiceUnit(w io.Writer, opts CaddyServiceUnitOptions) error {
	return caddyServiceTmpl.Execute(w, opts)
}
