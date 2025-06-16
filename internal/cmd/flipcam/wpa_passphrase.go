package flipcam

import (
	"fmt"
)

type wpaPassphraseFlag string

// String is used both by fmt.Print and by Cobra in help text
func (f *wpaPassphraseFlag) String() string {
	return string(*f)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (f *wpaPassphraseFlag) Set(v string) error {
	if len(v) < 8 {
		return fmt.Errorf("passphrase must be at least 8 characters long")
	}
	if len(v) > 63 {
		return fmt.Errorf("passphrase must be no more than 63 characters")
	}

	for _, r := range v {
		if r < 32 || r > 126 {
			return fmt.Errorf("passphrase must contain only printable ASCII characters")
		}
	}

	*f = wpaPassphraseFlag(v)
	return nil
}

// Type is only used in help text
func (f *wpaPassphraseFlag) Type() string {
	return "WpaPassphrase"
}
