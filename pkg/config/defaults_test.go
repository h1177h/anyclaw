package config

import "testing"

func TestDefaultConfigGatewayControlUIBasePath(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Gateway.ControlUI.BasePath != "/dashboard" {
		t.Fatalf("expected default control UI base path /dashboard, got %q", cfg.Gateway.ControlUI.BasePath)
	}
}
