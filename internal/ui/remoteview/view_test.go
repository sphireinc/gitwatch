package remoteview

import (
	"strings"
	"testing"
	"time"

	"github.com/jusanchez/gitwatch/internal/remotes"
)

func TestViewRendersRedactedRemoteStateAndPreservesSelection(t *testing.T) {
	now := time.Unix(200, 0)
	m := New(remotes.Dashboard{Now: now, StaleAfter: time.Minute, Remotes: []remotes.Remote{
		{Name: "origin", FetchURL: "https://example.test/repo.git", PushURL: "https://example.test/repo.git", Reachable: true, Default: true, LastFetchUnix: 1},
		{Name: "backup", FetchURL: "https://backup.test/repo.git", Reachable: false, LastError: "unreachable"},
	}})
	m.Move(1)
	m.SetDashboard(remotes.Dashboard{Now: now, StaleAfter: time.Minute, Remotes: m.Dashboard.Remotes})
	if m.Selected != 1 {
		t.Fatalf("selection = %d", m.Selected)
	}
	view := m.View()
	for _, want := range []string{"origin", "default", "backup", "unreachable", "stale", "error: unreachable"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q: %s", want, view)
		}
	}
}
