package provider

import (
	"context"
	"errors"
	"net/http"
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

func TestHTTPErrorDistinguishesPermissionFromRateLimit403(t *testing.T) {
	permission := &HTTPError{Status: http.StatusForbidden}
	if permission.IsRateLimited() || errors.Is(permission, ErrRateLimited) || Classify(context.Background(), permission) != StateUnauthorized {
		t.Fatalf("permission 403 classification: limited=%v unwrap=%v state=%s", permission.IsRateLimited(), errors.Is(permission, ErrRateLimited), Classify(context.Background(), permission))
	}
	quota := &HTTPError{Status: http.StatusForbidden, RateLimitRemaining: "0", RateLimitReset: "1730000000"}
	if !quota.IsRateLimited() || !errors.Is(quota, ErrRateLimited) || Classify(context.Background(), quota) != StateRateLimited {
		t.Fatalf("quota 403 classification: limited=%v unwrap=%v state=%s", quota.IsRateLimited(), errors.Is(quota, ErrRateLimited), Classify(context.Background(), quota))
	}
	retry := newHTTPError(&http.Response{StatusCode: http.StatusForbidden, Header: http.Header{"Retry-After": []string{"30"}, "X-Ratelimit-Remaining": []string{"12"}, "X-Ratelimit-Reset": []string{"1730000000"}}})
	if !retry.IsRateLimited() || retry.RetryAfter != "30" || retry.RateLimitRemaining != "12" || retry.RateLimitReset != "1730000000" {
		t.Fatalf("rate-limit metadata = %#v", retry)
	}
}
