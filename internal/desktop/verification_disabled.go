//go:build !opskat_e2e

package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

// VerificationTransport is absent from production builds.
func VerificationTransport() (application.Transport, error) { return nil, nil }
