package flipcamlib

import (
	"io"
	"strings"
	"text/template"
)

var sudoersConfTmpl = template.Must(template.New("sudoers.conf").Parse(`
{{- /* Trim leading spaces */ -}}
%flipcam ALL=NOPASSWD: {{.Commands}}
`))

func (f *FlipCam) GenerateSudoersConf(writer io.Writer) error {
	commands := []string{
		strings.Join(f.ipAddrAdd(), " "),
		strings.Join(f.ipAddrRemove(), " "),
		strings.Join(f.nmcliDisableManagedCmd(), " "),
		strings.Join(f.nmcliEnableManagedCmd(), " "),
	}

	if len(commands) == 0 {
		return nil
	}

	return sudoersConfTmpl.Execute(writer, map[string]interface{}{
		"Commands": strings.Join(commands, ", "),
	})
}
