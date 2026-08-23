package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/git"
)

func TestBuildRedactsDiagnosticMetadata(t *testing.T) {
	report := Build("1.0.0-dev", config.Defaults(), git.Discovery{Root: "/Users/private/repo", GitVersion: "git version 2.43\x1b[31m", Capabilities: git.Capabilities{Version: git.Version{Major: 2, Minor: 43}, Restore: true}}, "startup", &testError{value: "token=secret https://user:pass@example.test/repo\x1b"})
	data, _ := json.Marshal(report)
	text := string(data)
	if strings.Contains(text, "secret") || strings.Contains(text, "user:pass") || strings.Contains(text, "\x1b") {
		t.Fatalf("unsafe report: %q", text)
	}
	if report.Capabilities["restore"] != true || report.Correlation == "" {
		t.Fatalf("incomplete report: %#v", report)
	}
}

func TestWriteBundleIsPrivateAndBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "diagnostics.json")
	if err := WriteBundle(path, Report{Schema: 1, Correlation: "local-test"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
}

type testError struct{ value string }

func (e *testError) Error() string { return e.value }
