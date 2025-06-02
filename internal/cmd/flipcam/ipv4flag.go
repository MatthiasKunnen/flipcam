package flipcam

import (
	"fmt"
	"net/netip"
)

type ipv4Flag netip.Prefix

// String is used both by fmt.Print and by Cobra in help text
func (f *ipv4Flag) String() string {
	return netip.Prefix(*f).String()
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (f *ipv4Flag) Set(v string) error {
	prefix, err := netip.ParsePrefix(v)
	if err != nil {
		return err
	}

	if prefix.Bits() > 28 {
		return fmt.Errorf("prefix must be smaller than 28: %s", v)
	}

	if !prefix.Addr().Is4() {
		return fmt.Errorf("address must be IPv4: %s", v)
	}

	*f = ipv4Flag(prefix)
	return nil
}

// Type is only used in help text
func (f *ipv4Flag) Type() string {
	return "IPv4CIDR"
}

func (f *ipv4Flag) Prefix() netip.Prefix {
	return netip.Prefix(*f)
}
