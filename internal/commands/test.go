package commands

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jolehuit/clother/internal/profiles"
)

func runTest(ctx context.Context, c Context, args []string) (int, error) {
	var targets []profiles.Target
	if len(args) > 0 {
		target, err := profiles.Resolve(args[0], c.Catalog, c.Config)
		if err != nil {
			return 1, err
		}
		targets = []profiles.Target{target}
	} else {
		targets = profiles.All(c.Catalog, c.Config)
	}

	okCount, failCount := 0, 0
	for _, target := range targets {
		if target.Profile == "native" {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.TestURL, nil)
		if err != nil {
			fmt.Fprintf(c.Output.Stdout, "  %-18s invalid URL\n", target.Profile)
			failCount++
			continue
		}
		setCredentialHeader(req, target, targetCredential(target, c.Secrets))
		client, clientErr := clientForTarget(target, 5*time.Second)
		if clientErr != nil {
			fmt.Fprintf(c.Output.Stdout, "  %-18s invalid proxy\n", target.Profile)
			failCount++
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(c.Output.Stdout, "  %-18s unreachable\n", target.Profile)
			failCount++
			continue
		}
		_ = resp.Body.Close()
		fmt.Fprintf(c.Output.Stdout, "  %-18s reachable (HTTP %d)\n", target.Profile, resp.StatusCode)
		okCount++
	}
	fmt.Fprintf(c.Output.Stdout, "\nResults: %d reachable, %d failed\n", okCount, failCount)
	return 0, nil
}
