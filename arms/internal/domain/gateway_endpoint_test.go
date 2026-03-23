package domain

import "testing"

func TestNormalizeGatewayDriver_PicoClaw(t *testing.T) {
	for _, in := range []string{"picoclaw_ws", "PicoClaw", "pico-claw"} {
		if got := NormalizeGatewayDriver(in); got != GatewayDriverPicoClawWS {
			t.Fatalf("%q -> %q want %s", in, got, GatewayDriverPicoClawWS)
		}
	}
}

func TestNormalizeGatewayDriver_NullClawA2A(t *testing.T) {
	for _, in := range []string{"nullclaw_a2a", "nullclaw-http", "NULLCLAW_HTTP"} {
		if got := NormalizeGatewayDriver(in); got != GatewayDriverNullClawA2A {
			t.Fatalf("%q -> %q want %s", in, got, GatewayDriverNullClawA2A)
		}
	}
}

func TestNormalizeGatewayDriver_ZeroClaw(t *testing.T) {
	for _, in := range []string{"zeroclaw_ws", "ZeroClaw", "zero-claw"} {
		if got := NormalizeGatewayDriver(in); got != GatewayDriverZeroClawWS {
			t.Fatalf("%q -> %q want %s", in, got, GatewayDriverZeroClawWS)
		}
	}
}

func TestNormalizeGatewayDriver_Clawlet(t *testing.T) {
	for _, in := range []string{"clawlet_ws", "Clawlet", "clawlet-ws"} {
		if got := NormalizeGatewayDriver(in); got != GatewayDriverClawletWS {
			t.Fatalf("%q -> %q want %s", in, got, GatewayDriverClawletWS)
		}
	}
}

func TestNormalizeGatewayDriver_IronClaw(t *testing.T) {
	for _, in := range []string{"ironclaw_ws", "IronClaw", "iron-claw", "iron_claw"} {
		if got := NormalizeGatewayDriver(in); got != GatewayDriverIronClawWS {
			t.Fatalf("%q -> %q want %s", in, got, GatewayDriverIronClawWS)
		}
	}
}

func TestNormalizeGatewayDriver_MimiClaw(t *testing.T) {
	for _, in := range []string{"mimiclaw_ws", "MimiClaw", "mimi-claw"} {
		if got := NormalizeGatewayDriver(in); got != GatewayDriverMimiClawWS {
			t.Fatalf("%q -> %q want %s", in, got, GatewayDriverMimiClawWS)
		}
	}
}

func TestNormalizeGatewayDriver_NanobotCLI(t *testing.T) {
	for _, in := range []string{"nanobot_cli", "nanobot", "NANOBOT-CLI"} {
		if got := NormalizeGatewayDriver(in); got != GatewayDriverNanobotCLI {
			t.Fatalf("%q -> %q want %s", in, got, GatewayDriverNanobotCLI)
		}
	}
}

func TestNormalizeGatewayDriver_ZClawRelayHTTP(t *testing.T) {
	for _, in := range []string{"zclaw_relay_http", "zclaw", "ZCLAW-RELAY", "zclaw-http"} {
		if got := NormalizeGatewayDriver(in); got != GatewayDriverZClawRelayHTTP {
			t.Fatalf("%q -> %q want %s", in, got, GatewayDriverZClawRelayHTTP)
		}
	}
}
