package commands

import (
	"net/http"
	"net/url"
	"time"

	"github.com/jolehuit/clother/internal/profiles"
)

func clientForTarget(target profiles.Target, timeout time.Duration) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if target.ProxyURL != "" {
		proxyURL, err := url.Parse(target.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{Transport: transport, Timeout: timeout}, nil
}
