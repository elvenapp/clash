//go:build foss

package fs

import (
	"time"

	"clash-foss/constant/provider"
)

type ProviderFS interface {
	Read(providerType provider.ProviderType, providerName string) ([]byte, time.Time, error)
	Write(providerType provider.ProviderType, providerName string, data []byte) error
	Touch(providerType provider.ProviderType, providerName string) error
}
