package provider

import "testing"

func TestParseReleasesBounds(t *testing.T) {
	releases, err := ParseReleases([]byte(`[{"id":7,"name":"v1","tag_name":"v1.0.0","html_url":"https://github/release/7","draft":false,"prerelease":true}]`))
	if err != nil || len(releases) != 1 || releases[0].TagName != "v1.0.0" || !releases[0].Prerelease {
		t.Fatalf("releases = %#v, err=%v", releases, err)
	}
	if _, err := ParseReleases([]byte(`[{"id":0,"tag_name":"v1"}]`)); err == nil {
		t.Fatal("invalid release was accepted")
	}
}
