package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jolehuit/clother/internal/config"
	"github.com/jolehuit/clother/internal/profiles"
	"github.com/jolehuit/clother/internal/providers"
)

// IsHomebrew reports whether the running binary is managed by Homebrew.
//
// It first checks the HOMEBREW_PREFIX env var (set during `brew install`),
// then falls back to inspecting whether the resolved executable path lives
// inside a Homebrew Cellar directory — which is the case when the user runs
// a Homebrew-installed binary from their normal shell session.
func IsHomebrew() bool {
	if os.Getenv("HOMEBREW_PREFIX") != "" {
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	return strings.Contains(resolved, "/Cellar/")
}

func BuildEnv(target profiles.Target, secrets config.Secrets) ([]string, error) {
	envMap := map[string]string{}
	for _, pair := range os.Environ() {
		key, value, ok := splitEnv(pair)
		if ok {
			envMap[key] = value
		}
	}
	clearAnthropicEnv(envMap)

	if target.BaseURL != "" {
		envMap["ANTHROPIC_BASE_URL"] = target.BaseURL
	}
	if target.Model != "" {
		envMap["ANTHROPIC_MODEL"] = target.Model
		if target.Family == providers.FamilyLocal || target.Family == providers.FamilyCustomUnknown {
			envMap["ANTHROPIC_CUSTOM_MODEL_OPTION"] = target.Model
			envMap["ANTHROPIC_CUSTOM_MODEL_OPTION_NAME"] = target.DisplayName + ": " + target.Model
			envMap["ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION"] = "Custom model via " + target.DisplayName
		}
	}
	for key, value := range target.ModelTiers {
		switch key {
		case "haiku":
			envMap["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = value
		case "sonnet":
			envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"] = value
		case "opus":
			envMap["ANTHROPIC_DEFAULT_OPUS_MODEL"] = value
		case "fable":
			envMap["ANTHROPIC_DEFAULT_FABLE_MODEL"] = value
		case "small":
			envMap["ANTHROPIC_SMALL_FAST_MODEL"] = value
		}
	}

	switch target.AuthMode {
	case providers.AuthNone:
	case providers.AuthLiteral:
		envMap["ANTHROPIC_AUTH_TOKEN"] = target.LiteralAuthToken
		envMap["ANTHROPIC_API_KEY"] = ""
	case providers.AuthSecret:
		value := secrets[target.SecretKey]
		if value == "" {
			return nil, fmt.Errorf("%s not configured", target.SecretKey)
		}
		envMap["ANTHROPIC_AUTH_TOKEN"] = value
		if target.Family == providers.FamilyOpenRouter || target.Family == providers.FamilyLocal || target.Family == providers.FamilyCustomUnknown {
			envMap["ANTHROPIC_API_KEY"] = ""
		}
	case providers.AuthOptionalSecret:
		value := secrets[target.SecretKey]
		if value == "" {
			value = target.LiteralAuthToken
		}
		envMap["ANTHROPIC_AUTH_TOKEN"] = value
		envMap["ANTHROPIC_API_KEY"] = ""
	default:
		return nil, fmt.Errorf("unsupported auth mode %q", target.AuthMode)
	}

	if target.MaxContextTokens > 0 {
		envMap["CLAUDE_CODE_MAX_CONTEXT_TOKENS"] = fmt.Sprint(target.MaxContextTokens)
	}
	if target.AutoCompactWindow > 0 {
		envMap["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] = fmt.Sprint(target.AutoCompactWindow)
	}
	if target.SubagentModel != "" {
		envMap["CLAUDE_CODE_SUBAGENT_MODEL"] = target.SubagentModel
	}
	if target.GatewayDiscovery {
		envMap["CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY"] = "1"
	}
	if target.EffortLevel != "" {
		envMap["CLAUDE_CODE_EFFORT_LEVEL"] = target.EffortLevel
	}
	if len(target.ModelCapabilities) > 0 {
		capabilities := strings.Join(target.ModelCapabilities, ",")
		if target.Model != "" && (target.Family == providers.FamilyLocal || target.Family == providers.FamilyCustomUnknown) {
			envMap["ANTHROPIC_CUSTOM_MODEL_OPTION_SUPPORTED_CAPABILITIES"] = capabilities
		}
		for tier := range target.ModelTiers {
			switch tier {
			case "fable", "opus", "sonnet", "haiku":
				envMap["ANTHROPIC_DEFAULT_"+strings.ToUpper(tier)+"_MODEL_SUPPORTED_CAPABILITIES"] = capabilities
			}
		}
	}
	if target.ProxyURL != "" {
		for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
			envMap[key] = target.ProxyURL
		}
		noProxy := target.NoProxy
		if noProxy == "" {
			noProxy = "localhost,127.0.0.1,.local"
		}
		envMap["NO_PROXY"] = noProxy
		envMap["no_proxy"] = noProxy
	}

	return flattenEnv(envMap), nil
}

func clearAnthropicEnv(envMap map[string]string) {
	for key := range envMap {
		if strings.HasPrefix(key, "ANTHROPIC_") {
			delete(envMap, key)
		}
	}
}

func splitEnv(pair string) (string, string, bool) {
	for i := 0; i < len(pair); i++ {
		if pair[i] == '=' {
			return pair[:i], pair[i+1:], true
		}
	}
	return "", "", false
}

func flattenEnv(envMap map[string]string) []string {
	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, key+"="+value)
	}
	return env
}
