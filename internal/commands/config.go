package commands

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jolehuit/clother/internal/config"
	"github.com/jolehuit/clother/internal/launchers"
	"github.com/jolehuit/clother/internal/providers"
	"github.com/jolehuit/clother/internal/runtime"
)

var (
	validName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
)

func runConfig(_ context.Context, c Context, args []string) (int, error) {
	providerID := ""
	if len(args) > 0 {
		providerID = args[0]
	} else {
		var err error
		providerID, err = chooseProvider(c)
		if err != nil || providerID == "" {
			return 0, err
		}
	}

	switch providerID {
	case "openrouter":
		return configOpenRouter(c)
	case "custom":
		return configCustom(c)
	default:
		if provider, ok := c.Catalog.Get(providerID); ok {
			return configBuiltin(c, provider)
		}
		return 1, fmt.Errorf("unknown provider %q", providerID)
	}
}

func chooseProvider(c Context) (string, error) {
	index := 1
	choices := map[int]string{}
	c.Output.Header("Clother Configuration")
	for _, category := range c.Catalog.Categories() {
		fmt.Fprintln(c.Output.Stdout, category)
		for _, provider := range c.Catalog.ProvidersByCategory(category) {
			fmt.Fprintf(c.Output.Stdout, "  %2d. %-14s %s\n", index, provider.ID, provider.Description)
			choices[index] = provider.ID
			index++
		}
	}
	fmt.Fprintf(c.Output.Stdout, "  %2d. %-14s %s\n", index, "openrouter", "100+ models")
	choices[index] = "openrouter"
	index++
	fmt.Fprintf(c.Output.Stdout, "  %2d. %-14s %s\n", index, "custom", "Anthropic-compatible endpoint")
	choices[index] = "custom"

	answer, err := c.Prompt.Prompt("Choose provider number", "")
	if err != nil {
		return "", err
	}
	for number, providerID := range choices {
		if fmt.Sprint(number) == answer {
			return providerID, nil
		}
	}
	return "", fmt.Errorf("invalid choice %q", answer)
}

func configBuiltin(c Context, provider providers.Provider) (int, error) {
	if provider.AuthMode == providers.AuthSecret {
		current := c.Secrets[provider.KeyVar]
		if current != "" {
			fmt.Fprintf(c.Output.Stdout, "Current key: %s\n", config.MaskSecret(current))
		}
		label := "API key"
		if current != "" {
			label = "API key (empty to keep current)"
		}
		value, err := c.Prompt.PromptSecret(label)
		if err != nil {
			return 1, err
		}
		if strings.TrimSpace(value) != "" {
			c.Secrets[provider.KeyVar] = value
		}
	} else if provider.AuthMode == providers.AuthOptionalSecret {
		current := c.Secrets[provider.KeyVar]
		if current != "" {
			fmt.Fprintf(c.Output.Stdout, "Current token: %s\n", config.MaskSecret(current))
		}
		label := "API token (optional)"
		if current != "" {
			label = "API token (empty to keep, '-' to remove)"
		}
		value, err := c.Prompt.PromptSecret(label)
		if err != nil {
			return 1, err
		}
		switch strings.TrimSpace(value) {
		case "":
		case "-":
			delete(c.Secrets, provider.KeyVar)
		default:
			c.Secrets[provider.KeyVar] = value
		}
	}

	override := c.Config.ProviderOverrides[provider.ID]

	if provider.DefaultModel != "" {
		fmt.Fprintln(c.Output.Stdout, "Choose model:")
		for idx, choice := range provider.ModelChoices {
			fmt.Fprintf(c.Output.Stdout, "  %d. %-24s %s\n", idx+1, choice.ID, choice.Description)
		}
		defaultValue := provider.DefaultModel
		if override.Model != "" {
			defaultValue = override.Model
		}
		answer, err := c.Prompt.Prompt("Model", defaultValue)
		if err != nil {
			return 1, err
		}
		answer = resolveModelChoice(answer, provider.ModelChoices)
		if answer != "" && answer != provider.DefaultModel {
			override.Model = answer
		} else {
			override.Model = ""
		}
	}

	// Local backends may run on another machine (e.g. LM Studio on a LAN
	// host), so let the user point the launcher at a remote base URL.
	if provider.Family == providers.FamilyLocal {
		defaultURL := provider.BaseURL
		if override.BaseURL != "" {
			defaultURL = override.BaseURL
		}
		answer, err := c.Prompt.Prompt("Base URL", defaultURL)
		if err != nil {
			return 1, err
		}
		answer = strings.TrimSpace(answer)
		if answer != "" && answer != provider.BaseURL {
			if !strings.HasPrefix(answer, "http://") && !strings.HasPrefix(answer, "https://") {
				return 1, fmt.Errorf("invalid base URL %q (must start with http:// or https://)", answer)
			}
			override.BaseURL = strings.TrimRight(answer, "/")
		} else {
			override.BaseURL = ""
		}

		if err := promptProviderRuntimeOptions(c, &override.DefaultModel, &override.ModelTiers,
			&override.MaxContextTokens, &override.AutoCompactWindow, &override.ProxyURL,
			&override.NoProxy, &override.SubagentModel); err != nil {
			return 1, err
		}
	}

	if override.Empty() {
		delete(c.Config.ProviderOverrides, provider.ID)
	} else {
		c.Config.ProviderOverrides[provider.ID] = override
	}
	return persistConfig(c)
}

func configOpenRouter(c Context) (int, error) {
	current := c.Secrets["OPENROUTER_API_KEY"]
	if current != "" {
		fmt.Fprintf(c.Output.Stdout, "Current key: %s\n", config.MaskSecret(current))
	}
	value, err := c.Prompt.PromptSecret("OpenRouter API key (empty to keep current)")
	if err != nil {
		return 1, err
	}
	if strings.TrimSpace(value) != "" {
		c.Secrets["OPENROUTER_API_KEY"] = value
	}
	for {
		model, err := c.Prompt.Prompt("Model ID (empty to stop)", "")
		if err != nil {
			return 1, err
		}
		if strings.TrimSpace(model) == "" {
			break
		}
		name, err := c.Prompt.Prompt("Alias", defaultAliasName(model))
		if err != nil {
			return 1, err
		}
		if !validName.MatchString(name) {
			return 1, fmt.Errorf("invalid alias %q (use lowercase letters, digits, \"-\" or \"_\")", name)
		}
		c.Config.OpenRouterAliases[name] = model
	}
	return persistConfig(c)
}

func configCustom(c Context) (int, error) {
	name, err := c.Prompt.Prompt("Provider name", "")
	if err != nil {
		return 1, err
	}
	if !validName.MatchString(name) {
		return 1, fmt.Errorf("invalid provider name %q", name)
	}

	existing := c.Config.CustomProviders[name]

	urlLabel := "Base URL"
	if existing.BaseURL != "" {
		fmt.Fprintf(c.Output.Stdout, "Current URL: %s\n", existing.BaseURL)
		urlLabel = "Base URL (empty to keep current)"
	}
	baseURL, err := c.Prompt.Prompt(urlLabel, "")
	if err != nil {
		return 1, err
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = existing.BaseURL
	}
	if baseURL == "" {
		return 1, fmt.Errorf("base URL is required")
	}

	defaultModel := existing.DefaultModel
	modelTiers := existing.ModelTiers
	maxContextTokens := existing.MaxContextTokens
	autoCompactWindow := existing.AutoCompactWindow
	proxyURL := existing.ProxyURL
	noProxy := existing.NoProxy
	subagentModel := existing.SubagentModel
	if err := promptProviderRuntimeOptions(c, &defaultModel, &modelTiers, &maxContextTokens,
		&autoCompactWindow, &proxyURL, &noProxy, &subagentModel); err != nil {
		return 1, err
	}

	keyVar := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_API_KEY"
	current := c.Secrets[keyVar]
	keyLabel := "API key"
	if current != "" {
		fmt.Fprintf(c.Output.Stdout, "Current key: %s\n", config.MaskSecret(current))
		keyLabel = "API key (empty to keep current)"
	}
	apiKey, err := c.Prompt.PromptSecret(keyLabel)
	if err != nil {
		return 1, err
	}
	if strings.TrimSpace(apiKey) != "" {
		c.Secrets[keyVar] = apiKey
	}

	c.Config.CustomProviders[name] = config.CustomProvider{
		Name:              name,
		DisplayName:       name,
		BaseURL:           baseURL,
		APIKeyEnv:         keyVar,
		DefaultModel:      defaultModel,
		ModelTiers:        modelTiers,
		MaxContextTokens:  maxContextTokens,
		AutoCompactWindow: autoCompactWindow,
		ProxyURL:          proxyURL,
		NoProxy:           noProxy,
		SubagentModel:     subagentModel,
	}
	return persistConfig(c)
}

func promptProviderRuntimeOptions(c Context, defaultModel *string, modelTiers *map[string]string,
	maxContextTokens, autoCompactWindow *int, proxyURL, noProxy, subagentModel *string) error {
	value, err := promptOptional(c, "Default model (shown as a custom /model option)", *defaultModel)
	if err != nil {
		return err
	}
	*defaultModel = value

	if *modelTiers == nil {
		*modelTiers = map[string]string{}
	}
	for _, tier := range []string{"fable", "opus", "sonnet", "haiku"} {
		label := strings.ToUpper(tier[:1]) + tier[1:] + " model (optional)"
		value, err := promptOptional(c, label, (*modelTiers)[tier])
		if err != nil {
			return err
		}
		if value == "" {
			delete(*modelTiers, tier)
		} else {
			(*modelTiers)[tier] = value
		}
	}

	if *maxContextTokens, err = promptTokenCount(c, "Maximum context tokens (optional)", *maxContextTokens, 1, 1_000_000); err != nil {
		return err
	}
	if *autoCompactWindow, err = promptTokenCount(c, "Auto-compact window (optional, 100000-1000000)", *autoCompactWindow, 100_000, 1_000_000); err != nil {
		return err
	}
	if *autoCompactWindow > 0 && *maxContextTokens > 0 && *autoCompactWindow > *maxContextTokens {
		return fmt.Errorf("auto-compact window (%d) cannot exceed maximum context (%d)", *autoCompactWindow, *maxContextTokens)
	}

	if *proxyURL, err = promptOptional(c, "HTTP(S) proxy URL (optional)", *proxyURL); err != nil {
		return err
	}
	if *proxyURL != "" {
		if !strings.HasPrefix(*proxyURL, "http://") && !strings.HasPrefix(*proxyURL, "https://") {
			return fmt.Errorf("invalid proxy URL %q (must start with http:// or https://)", *proxyURL)
		}
		if *noProxy, err = promptOptional(c, "NO_PROXY", firstNonEmpty(*noProxy, "localhost,127.0.0.1,.local")); err != nil {
			return err
		}
	} else {
		*noProxy = ""
	}

	if *subagentModel, err = promptOptional(c, "Subagent model (optional; use 'inherit' on one GPU)", *subagentModel); err != nil {
		return err
	}
	return nil
}

func promptOptional(c Context, label, current string) (string, error) {
	if current != "" {
		label += " ('-' to clear)"
	}
	answer, err := c.Prompt.Prompt(label, current)
	if err != nil {
		return "", err
	}
	answer = strings.TrimSpace(answer)
	if answer == "-" {
		return "", nil
	}
	return answer, nil
}

func promptTokenCount(c Context, label string, current, min, max int) (int, error) {
	defaultValue := ""
	if current > 0 {
		defaultValue = strconv.Itoa(current)
		label += " ('-' to clear)"
	}
	answer, err := c.Prompt.Prompt(label, defaultValue)
	if err != nil {
		return 0, err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" || answer == "-" || answer == "0" {
		return 0, nil
	}
	value, err := strconv.Atoi(answer)
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("invalid token count %q (must be %d-%d)", answer, min, max)
	}
	return value, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func persistConfig(c Context) (int, error) {
	config.NormalizeLegacySecrets(c.Secrets, c.Catalog)
	if err := config.SaveConfig(c.Paths.ConfigFile, c.Config); err != nil {
		return 1, err
	}
	if err := config.SaveSecrets(c.Paths.SecretsFile, c.Secrets); err != nil {
		return 1, err
	}
	execPath, execErr := os.Executable()
	if execErr != nil {
		return 1, execErr
	}
	if err := launchers.Sync(execPath, c.Paths, c.Catalog, c.Config, runtime.IsHomebrew()); err != nil {
		return 1, err
	}
	c.Output.Success("configuration saved")
	return 0, nil
}

func defaultAliasName(model string) string {
	model = strings.ToLower(model)
	if slash := strings.LastIndex(model, "/"); slash >= 0 {
		model = model[slash+1:]
	}
	// Model IDs may carry characters that are invalid in an alias (used as
	// launcher name), e.g. the ":free"/":exacto" variant suffixes. Map anything
	// outside the alias charset to "-" so the suggested default always passes
	// validName.
	var b strings.Builder
	for _, r := range model {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	name := b.String()
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return strings.Trim(name, "-")
}

func resolveModelChoice(answer string, choices []providers.ModelChoice) string {
	answer = strings.TrimSpace(answer)
	if idx, err := strconv.Atoi(answer); err == nil {
		if idx >= 1 && idx <= len(choices) {
			return choices[idx-1].ID
		}
	}
	return answer
}
