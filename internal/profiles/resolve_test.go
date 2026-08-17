package profiles

import (
	"testing"

	"github.com/jolehuit/clother/internal/config"
	"github.com/jolehuit/clother/internal/providers"
)

func TestResolveAppliesBaseURLOverride(t *testing.T) {
	t.Parallel()

	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.File{
		Version: 1,
		ProviderOverrides: map[string]config.ProviderOverride{
			"lmstudio": {BaseURL: "http://192.168.123.123:1234"},
		},
		OpenRouterAliases: map[string]string{},
		CustomProviders:   map[string]config.CustomProvider{},
	}

	target, err := Resolve("lmstudio", catalog, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if target.BaseURL != "http://192.168.123.123:1234" {
		t.Fatalf("BaseURL = %q, want the override", target.BaseURL)
	}
	if target.TestURL != "http://192.168.123.123:1234" {
		t.Fatalf("TestURL = %q, want the override", target.TestURL)
	}

	// Without an override the catalog URL is untouched.
	plain, err := Resolve("ollama", catalog, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if plain.BaseURL != "http://localhost:11434" {
		t.Fatalf("ollama BaseURL = %q, want catalog default", plain.BaseURL)
	}
}

func TestResolveCombinesModelAndBaseURLOverrides(t *testing.T) {
	t.Parallel()

	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.File{
		Version: 1,
		ProviderOverrides: map[string]config.ProviderOverride{
			"zai": {Model: "glm-4.7", BaseURL: "https://proxy.example.com/anthropic"},
		},
		OpenRouterAliases: map[string]string{},
		CustomProviders:   map[string]config.CustomProvider{},
	}

	target, err := Resolve("zai", catalog, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if target.Model != "glm-4.7" {
		t.Fatalf("Model = %q, want glm-4.7", target.Model)
	}
	if target.BaseURL != "https://proxy.example.com/anthropic" {
		t.Fatalf("BaseURL = %q, want the override", target.BaseURL)
	}
}

func TestResolveCustomProviderAllowsOptionalAuthenticationAndRuntimeOptions(t *testing.T) {
	t.Parallel()
	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}
	discovery := true
	cfg := &config.File{
		Version:           1,
		ProviderOverrides: map[string]config.ProviderOverride{},
		OpenRouterAliases: map[string]string{},
		CustomProviders: map[string]config.CustomProvider{
			"local-lab": {
				Name:              "local-lab",
				DisplayName:       "Local Lab",
				BaseURL:           "http://localhost:1234",
				APIKeyEnv:         "LOCAL_LAB_API_KEY",
				DefaultModel:      "qwen-local",
				GatewayDiscovery:  &discovery,
				EffortLevel:       "medium",
				ModelCapabilities: []string{"effort"},
			},
		},
	}
	target, err := Resolve("local-lab", catalog, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if target.AuthMode != providers.AuthOptionalSecret {
		t.Fatalf("AuthMode = %q, want optional secret", target.AuthMode)
	}
	if !target.GatewayDiscovery || target.EffortLevel != "medium" {
		t.Fatalf("runtime options not resolved: %+v", target)
	}
}
