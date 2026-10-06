package config

import "testing"

func TestDevLoginForbiddenInProduction(t *testing.T) {
	t.Setenv("YU_ENV", "production")
	t.Setenv("YU_DEV_LOGIN", "true")
	if _, err := Load(); err == nil {
		t.Fatal("expected production dev-login configuration to fail")
	}
}

func TestDevLoginAllowedInDevelopment(t *testing.T) {
	t.Setenv("YU_ENV", "development")
	t.Setenv("YU_DEV_LOGIN", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DevLogin {
		t.Fatal("expected dev-login to be enabled")
	}
}
