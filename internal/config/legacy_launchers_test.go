package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jolehuit/clother/internal/providers"
)

func TestMigrateLegacyLaunchersIgnoresShellVariableBaseURL(t *testing.T) {
	t.Parallel()
	binDir := t.TempDir()
	script := "#!/bin/sh\nreadonly BASE_URL=https://example.com\nexport ANTHROPIC_BASE_URL=\"${BASE_URL}\"\nexport ANTHROPIC_MODEL=qwen\n"
	if err := os.WriteFile(filepath.Join(binDir, "clother-lmstudio-models"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg := &File{Version: 1, ProviderOverrides: map[string]ProviderOverride{}, OpenRouterAliases: map[string]string{}, CustomProviders: map[string]CustomProvider{}}
	if err := MigrateLegacyLaunchers(binDir, catalog, cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.CustomProviders["lmstudio-models"]; ok {
		t.Fatal("utility script with an unexpanded URL was migrated as a provider")
	}
}

func TestMigrateLegacyLaunchersKeepsFixedHTTPURL(t *testing.T) {
	t.Parallel()
	binDir := t.TempDir()
	script := "#!/bin/sh\nexport ANTHROPIC_BASE_URL=https://example.com/anthropic\nexport ANTHROPIC_MODEL=qwen\n"
	if err := os.WriteFile(filepath.Join(binDir, "clother-lab"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg := &File{Version: 1, ProviderOverrides: map[string]ProviderOverride{}, OpenRouterAliases: map[string]string{}, CustomProviders: map[string]CustomProvider{}}
	if err := MigrateLegacyLaunchers(binDir, catalog, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.CustomProviders["lab"].BaseURL != "https://example.com/anthropic" {
		t.Fatalf("fixed legacy URL was not migrated: %+v", cfg.CustomProviders)
	}
}
