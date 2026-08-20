package provider

import (
	"errors"
	"testing"
)

func TestParseGitHubRemote(t *testing.T) {
	for _, raw := range []string{"git@github.com:owner/project.git", "https://github.com/owner/project"} {
		repository, ok := ParseGitHubRemote(raw)
		if !ok || repository.Owner != "owner" || repository.Name != "project" {
			t.Fatalf("unexpected parse for %q: %#v", raw, repository)
		}
	}
	if _, ok := ParseGitHubRemote("https://gitlab.com/owner/project"); ok {
		t.Fatal("non-GitHub remote was detected")
	}
}

func TestEnvironmentTokenDoesNotExposeMissingSecret(t *testing.T) {
	_, err := EnvironmentToken("GITWATCH_TEST_TOKEN_MISSING").Token()
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected missing token, got %v", err)
	}
}

func TestCLITokenFailureDoesNotExposeCommandDetails(t *testing.T) {
	_, err := CLIToken{Binary: "gitwatch-test-missing-gh"}.Token()
	if !errors.Is(err, ErrNoToken) {
		t.Fatalf("expected missing token, got %v", err)
	}
}
