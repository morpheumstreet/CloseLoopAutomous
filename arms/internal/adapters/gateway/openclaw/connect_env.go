package openclaw

// ConnectEnv is process-wide configuration for the OpenClaw operator connect handshake,
// applied to every pooled WebSocket client (openclaw_ws, zeroclaw_ws, nemoclaw_ws, …).
type ConnectEnv struct {
	DeviceSigning      bool
	DeviceIdentityFile string
}

// MergeInto copies fields into [Options] (typically gateway endpoint URL/token/session still come from the row).
func (e ConnectEnv) MergeInto(o Options) Options {
	o.DeviceSigning = e.DeviceSigning
	o.DeviceIdentityFile = e.DeviceIdentityFile
	return o
}
