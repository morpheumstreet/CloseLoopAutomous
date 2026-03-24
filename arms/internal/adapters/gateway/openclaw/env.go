package openclaw

import (
	"os"
	"strings"
)

// truthyEnv matches common ARMS_* "enabled" conventions.
func truthyEnv(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

// ApplyArmsDeviceEnv sets DeviceSigning / DeviceIdentityFile from ARMS_DEVICE_SIGNING and
// ARMS_DEVICE_IDENTITY_FILE when those environment variables are set.
func ApplyArmsDeviceEnv(opts *Options) {
	if opts == nil {
		return
	}
	if truthyEnv(os.Getenv("ARMS_DEVICE_SIGNING")) {
		opts.DeviceSigning = true
	}
	if v := strings.TrimSpace(os.Getenv("ARMS_DEVICE_IDENTITY_FILE")); v != "" {
		opts.DeviceIdentityFile = v
	}
}
