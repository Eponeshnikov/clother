package commands

import (
	"context"
	"fmt"
	"strings"
)

func runRemove(_ context.Context, c Context, args []string) (int, error) {
	if len(args) != 1 {
		return 1, fmt.Errorf("usage: clother remove <profile>")
	}

	name := strings.TrimSpace(args[0])
	description, ok := removableProfileDescription(c, name)
	if !ok {
		return 1, fmt.Errorf("unknown profile %q", name)
	}
	if !c.Options.Yes {
		confirmed, err := c.Prompt.Confirm(description+"?", false)
		if err != nil {
			return 1, err
		}
		if !confirmed {
			return 0, nil
		}
	}

	result, err := removeProfile(c, name)
	if err != nil {
		return 1, err
	}
	code, err := persistConfig(c)
	if err != nil {
		return code, err
	}
	c.Output.Success(result)
	return 0, nil
}

func removableProfileDescription(c Context, name string) (string, bool) {
	if _, ok := c.Config.CustomProviders[name]; ok {
		return fmt.Sprintf("Remove custom profile %q", name), true
	}
	alias := strings.TrimPrefix(name, "or-")
	if _, ok := c.Config.OpenRouterAliases[alias]; ok {
		return fmt.Sprintf("Remove OpenRouter profile %q", "or-"+alias), true
	}
	if _, ok := c.Catalog.Get(name); ok {
		return fmt.Sprintf("Reset built-in provider %q to defaults and remove its saved API key", name), true
	}
	return "", false
}

func removeProfile(c Context, name string) (string, error) {
	if custom, ok := c.Config.CustomProviders[name]; ok {
		delete(c.Config.CustomProviders, name)
		if custom.APIKeyEnv != "" {
			delete(c.Secrets, custom.APIKeyEnv)
		}
		return fmt.Sprintf("custom profile %q removed", name), nil
	}

	alias := strings.TrimPrefix(name, "or-")
	if _, ok := c.Config.OpenRouterAliases[alias]; ok {
		delete(c.Config.OpenRouterAliases, alias)
		return fmt.Sprintf("OpenRouter profile %q removed", "or-"+alias), nil
	}

	provider, ok := c.Catalog.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown profile %q", name)
	}
	delete(c.Config.ProviderOverrides, name)
	if provider.KeyVar != "" {
		delete(c.Secrets, provider.KeyVar)
	}
	return fmt.Sprintf("built-in provider %q reset to defaults", name), nil
}
