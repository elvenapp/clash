//go:build foss

package provider

import (
	"clash-foss/constant"
)

// Vehicle Type
const (
	File       VehicleType = "File"
	HTTP       VehicleType = "HTTP"
	Compatible VehicleType = "Compatible"
)

// VehicleType defined
type VehicleType string

func (v VehicleType) String() string {
	return string(v)
}

func (v *VehicleType) MarshalBinary() (data []byte, err error) {
	return []byte(*v), nil
}

func (v *VehicleType) UnmarshalBinary(data []byte) error {
	*v = VehicleType(data)

	return nil
}

type Vehicle interface {
	Read() ([]byte, error)
	Type() VehicleType
}

// Provider Type
const (
	Proxy ProviderType = "Proxy"
	Rule  ProviderType = "Rule"
)

// ProviderType defined
type ProviderType string

func (pt ProviderType) String() string {
	return string(pt)
}

func (pt *ProviderType) MarshalBinary() (data []byte, err error) {
	return []byte(*pt), err
}

func (pt *ProviderType) UnmarshalBinary(data []byte) error {
	*pt = ProviderType(data)

	return nil
}

// Provider interface
type Provider interface {
	Name() string
	VehicleType() VehicleType
	Type() ProviderType
	Initial() error
	Update() error
}

// ProxyProvider interface
type ProxyProvider interface {
	Provider
	Proxies() []constant.Proxy
	// Touch is used to inform the provider that the proxy is actually being used while getting the list of proxies.
	// Commonly used in DialContext and DialPacketConn
	Touch()
	HealthCheck()
	HealthCheckElement(name string)
}

// Rule Type
const (
	Domain RuleType = iota
	IPCIDR
	Classical
)

// RuleType defined
type RuleType int

func (rt RuleType) String() string {
	switch rt {
	case Domain:
		return "Domain"
	case IPCIDR:
		return "IPCIDR"
	case Classical:
		return "Classical"
	default:
		return "Unknown"
	}
}

// RuleProvider interface
type RuleProvider interface {
	Provider
	Behavior() RuleType
	Match(*constant.Metadata) bool
	ShouldResolveIP() bool
	AsRule(adaptor string) constant.Rule
}
