package commands

import (
	"testing"

	"github.com/jolehuit/clother/internal/config"
	"github.com/jolehuit/clother/internal/providers"
)

func removeTestContext(t *testing.T) Context {
	t.Helper()
	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}
	return Context{
		Config: &config.File{
			Version:           1,
			ProviderOverrides: map[string]config.ProviderOverride{},
			OpenRouterAliases: map[string]string{},
			CustomProviders:   map[string]config.CustomProvider{},
		},
		Secrets: config.Secrets{},
		Catalog: catalog,
	}
}

func TestRemoveProfileDeletesCustomConfigAndSecret(t *testing.T) {
	c := removeTestContext(t)
	c.Config.CustomProviders["lmstudio-remote"] = config.CustomProvider{
		Name:      "lmstudio-remote",
		APIKeyEnv: "LMSTUDIO_REMOTE_API_KEY",
	}
	c.Secrets["LMSTUDIO_REMOTE_API_KEY"] = "secret"

	if _, err := removeProfile(c, "lmstudio-remote"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Config.CustomProviders["lmstudio-remote"]; ok {
		t.Fatal("custom profile was not removed")
	}
	if _, ok := c.Secrets["LMSTUDIO_REMOTE_API_KEY"]; ok {
		t.Fatal("custom profile secret was not removed")
	}
}

func TestRemoveProfileResetsBuiltinOverrideAndSecret(t *testing.T) {
	c := removeTestContext(t)
	c.Config.ProviderOverrides["lmstudio"] = config.ProviderOverride{BaseURL: "http://remote:1234"}
	c.Secrets["LMSTUDIO_API_KEY"] = "secret"

	if _, err := removeProfile(c, "lmstudio"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Config.ProviderOverrides["lmstudio"]; ok {
		t.Fatal("built-in override was not removed")
	}
	if _, ok := c.Secrets["LMSTUDIO_API_KEY"]; ok {
		t.Fatal("built-in provider secret was not removed")
	}
	if _, ok := c.Catalog.Get("lmstudio"); !ok {
		t.Fatal("built-in provider disappeared from catalog")
	}
}

func TestRemoveProfileAcceptsOpenRouterLauncherName(t *testing.T) {
	c := removeTestContext(t)
	c.Config.OpenRouterAliases["qwen"] = "qwen/qwen3"

	if _, err := removeProfile(c, "or-qwen"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Config.OpenRouterAliases["qwen"]; ok {
		t.Fatal("OpenRouter alias was not removed")
	}
}
