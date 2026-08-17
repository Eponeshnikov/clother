package update

import (
	"context"
	"testing"
)

func TestDownloadLatestIfNewerSkipsLocalBuilds(t *testing.T) {
	t.Parallel()
	path, version, cleanup, err := DownloadLatestIfNewer(context.Background(), "3.0.10-local.2")
	if err != nil {
		t.Fatal(err)
	}
	if path != "" || version != "" || cleanup != nil {
		t.Fatalf("local build attempted update: path=%q version=%q cleanup=%v", path, version, cleanup != nil)
	}
}
