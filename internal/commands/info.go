package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jolehuit/clother/internal/profiles"
)

func runInfo(_ context.Context, c Context, args []string) (int, error) {
	if len(args) == 0 {
		return 1, fmt.Errorf("usage: clother info <provider>")
	}
	target, err := profiles.Resolve(args[0], c.Catalog, c.Config)
	if err != nil {
		return 1, err
	}
	if c.Options.Format == "json" {
		data, _ := json.MarshalIndent(target, "", "  ")
		fmt.Fprintln(c.Output.Stdout, string(data))
		return 0, nil
	}
	c.Output.Header("Provider Info")
	fmt.Fprintf(c.Output.Stdout, "Profile:     %s\n", target.Profile)
	fmt.Fprintf(c.Output.Stdout, "Name:        %s\n", target.DisplayName)
	fmt.Fprintf(c.Output.Stdout, "Family:      %s\n", target.Family)
	fmt.Fprintf(c.Output.Stdout, "Base URL:    %s\n", target.BaseURL)
	if target.Model != "" {
		fmt.Fprintf(c.Output.Stdout, "Model:       %s\n", target.Model)
	}
	for _, tier := range []string{"fable", "opus", "sonnet", "haiku"} {
		if model := target.ModelTiers[tier]; model != "" {
			label := strings.ToUpper(tier[:1]) + tier[1:] + ":"
			fmt.Fprintf(c.Output.Stdout, "%-12s %s\n", label, model)
		}
	}
	if target.MaxContextTokens > 0 {
		fmt.Fprintf(c.Output.Stdout, "Context:     %d tokens\n", target.MaxContextTokens)
	}
	if target.AutoCompactWindow > 0 {
		fmt.Fprintf(c.Output.Stdout, "Auto-compact: %d tokens\n", target.AutoCompactWindow)
	}
	if target.ProxyURL != "" {
		fmt.Fprintf(c.Output.Stdout, "Proxy:       %s\n", target.ProxyURL)
	}
	if target.SubagentModel != "" {
		fmt.Fprintf(c.Output.Stdout, "Subagents:   %s\n", target.SubagentModel)
	}
	if target.GatewayDiscovery {
		fmt.Fprintln(c.Output.Stdout, "Discovery:   /v1/models")
	}
	if target.EffortLevel != "" {
		fmt.Fprintf(c.Output.Stdout, "Effort:      %s\n", target.EffortLevel)
	}
	if len(target.ModelCapabilities) > 0 {
		fmt.Fprintf(c.Output.Stdout, "Capabilities: %s\n", strings.Join(target.ModelCapabilities, ","))
	}
	if target.SecretKey != "" {
		status := "configured"
		if c.Secrets[target.SecretKey] == "" {
			status = "not configured"
		}
		fmt.Fprintf(c.Output.Stdout, "Credential:  %s (%s)\n", target.SecretKey, status)
	}
	return 0, nil
}
