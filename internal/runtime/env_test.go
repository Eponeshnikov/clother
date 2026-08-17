package runtime

import (
	"strings"
	"testing"

	"github.com/jolehuit/clother/internal/config"
	"github.com/jolehuit/clother/internal/profiles"
	"github.com/jolehuit/clother/internal/providers"
)

func TestBuildEnvForOpenRouter(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:   "or-kimi",
		Family:    providers.FamilyOpenRouter,
		BaseURL:   "https://openrouter.ai/api",
		AuthMode:  providers.AuthSecret,
		SecretKey: "OPENROUTER_API_KEY",
		ModelTiers: map[string]string{
			"haiku":  "moonshotai/kimi-k2.5",
			"sonnet": "moonshotai/kimi-k2.5",
			"opus":   "moonshotai/kimi-k2.5",
		},
	}

	env, err := BuildEnv(target, config.Secrets{"OPENROUTER_API_KEY": "sk-openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(env, "\n")
	for _, expected := range []string{
		"ANTHROPIC_BASE_URL=https://openrouter.ai/api",
		"ANTHROPIC_AUTH_TOKEN=sk-openrouter",
		"ANTHROPIC_API_KEY=",
		"ANTHROPIC_DEFAULT_OPUS_MODEL=moonshotai/kimi-k2.5",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("env missing %q:\n%s", expected, text)
		}
	}
}

func TestBuildEnvCustomProviderClearsAPIKey(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:   "myprovider",
		Family:    providers.FamilyCustomUnknown,
		BaseURL:   "https://api.example.com/anthropic",
		AuthMode:  providers.AuthSecret,
		SecretKey: "MYPROVIDER_API_KEY",
	}

	env, err := BuildEnv(target, config.Secrets{"MYPROVIDER_API_KEY": "sk-custom"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if got["ANTHROPIC_AUTH_TOKEN"] != "sk-custom" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN = %q, want sk-custom", got["ANTHROPIC_AUTH_TOKEN"])
	}
	if v, ok := got["ANTHROPIC_API_KEY"]; !ok || v != "" {
		t.Fatalf("ANTHROPIC_API_KEY should be cleared for custom providers, got %q (present=%v)", v, ok)
	}
}

func TestBuildEnvFailsWhenSecretMissing(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:   "zai",
		Family:    providers.FamilyAnthropicCompatibleNonClaude,
		AuthMode:  providers.AuthSecret,
		SecretKey: "ZAI_API_KEY",
		BaseURL:   "https://api.z.ai/api/anthropic",
	}
	if _, err := BuildEnv(target, config.Secrets{}); err == nil {
		t.Fatal("expected missing secret error")
	}
}

func TestBuildEnvNativeClearsInheritedAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "https://evil.example")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "bad-token")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "bad-model")

	target := profiles.Target{
		Profile:  "native",
		Family:   providers.FamilyClaudeStrict,
		AuthMode: providers.AuthNone,
	}

	env, err := BuildEnv(target, config.Secrets{})
	if err != nil {
		t.Fatal(err)
	}
	for key := range envToMap(env) {
		if strings.HasPrefix(key, "ANTHROPIC_") {
			t.Fatalf("native launcher leaked %s from parent environment", key)
		}
	}
}

func TestBuildEnvClearsUnusedTierVariables(t *testing.T) {
	t.Setenv("ANTHROPIC_DEFAULT_HAIKU_MODEL", "stale-haiku")
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "stale-sonnet")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "stale-opus")

	target := profiles.Target{
		Profile:   "zai",
		Family:    providers.FamilyAnthropicCompatibleNonClaude,
		BaseURL:   "https://api.z.ai/api/anthropic",
		AuthMode:  providers.AuthSecret,
		SecretKey: "ZAI_API_KEY",
		ModelTiers: map[string]string{
			"opus": "glm-5",
		},
	}

	env, err := BuildEnv(target, config.Secrets{"ZAI_API_KEY": "sk-zai"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if got["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "glm-5" {
		t.Fatalf("unexpected opus model: %q", got["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if _, ok := got["ANTHROPIC_DEFAULT_HAIKU_MODEL"]; ok {
		t.Fatal("haiku tier leaked from inherited environment")
	}
	if _, ok := got["ANTHROPIC_DEFAULT_SONNET_MODEL"]; ok {
		t.Fatal("sonnet tier leaked from inherited environment")
	}
}

func TestBuildEnvForAuthenticatedProxiedLocalModel(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:           "lmstudio",
		DisplayName:       "LM Studio",
		Family:            providers.FamilyLocal,
		BaseURL:           "https://lmstudio.example.com",
		Model:             "qwen3.8-27b-mtp",
		ModelTiers:        map[string]string{"opus": "qwen3.8-27b-mtp", "sonnet": "qwopus3.6-27b-v2-mtp"},
		AuthMode:          providers.AuthOptionalSecret,
		SecretKey:         "LMSTUDIO_API_KEY",
		LiteralAuthToken:  "lmstudio",
		MaxContextTokens:  110_000,
		AutoCompactWindow: 100_000,
		ProxyURL:          "http://127.0.0.1:2080",
		SubagentModel:     "inherit",
		GatewayDiscovery:  true,
		EffortLevel:       "high",
		ModelCapabilities: []string{"effort", "thinking"},
	}

	env, err := BuildEnv(target, config.Secrets{"LMSTUDIO_API_KEY": "lms-secret"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	want := map[string]string{
		"ANTHROPIC_AUTH_TOKEN":                                  "lms-secret",
		"ANTHROPIC_MODEL":                                       "qwen3.8-27b-mtp",
		"ANTHROPIC_CUSTOM_MODEL_OPTION":                         "qwen3.8-27b-mtp",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":                          "qwen3.8-27b-mtp",
		"ANTHROPIC_DEFAULT_SONNET_MODEL":                        "qwopus3.6-27b-v2-mtp",
		"CLAUDE_CODE_MAX_CONTEXT_TOKENS":                        "110000",
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW":                       "100000",
		"CLAUDE_CODE_SUBAGENT_MODEL":                            "inherit",
		"HTTP_PROXY":                                            "http://127.0.0.1:2080",
		"HTTPS_PROXY":                                           "http://127.0.0.1:2080",
		"CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY":            "1",
		"CLAUDE_CODE_EFFORT_LEVEL":                              "high",
		"ANTHROPIC_CUSTOM_MODEL_OPTION_SUPPORTED_CAPABILITIES":  "effort,thinking",
		"ANTHROPIC_DEFAULT_OPUS_MODEL_SUPPORTED_CAPABILITIES":   "effort,thinking",
		"ANTHROPIC_DEFAULT_SONNET_MODEL_SUPPORTED_CAPABILITIES": "effort,thinking",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, want %q", key, got[key], value)
		}
	}
}

func TestBuildEnvOptionalSecretFallsBackToLiteralToken(t *testing.T) {
	t.Parallel()
	target := profiles.Target{
		Family:           providers.FamilyLocal,
		AuthMode:         providers.AuthOptionalSecret,
		SecretKey:        "LMSTUDIO_API_KEY",
		LiteralAuthToken: "lmstudio",
	}
	env, err := BuildEnv(target, config.Secrets{})
	if err != nil {
		t.Fatal(err)
	}
	if got := envToMap(env)["ANTHROPIC_AUTH_TOKEN"]; got != "lmstudio" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN = %q, want literal fallback", got)
	}
}

func envToMap(env []string) map[string]string {
	out := map[string]string{}
	for _, pair := range env {
		key, value, ok := splitEnv(pair)
		if ok {
			out[key] = value
		}
	}
	return out
}
