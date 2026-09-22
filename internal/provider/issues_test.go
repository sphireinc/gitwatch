package provider

import "testing"

func TestParseIssuesBoundsAndExcludesPullRequests(t *testing.T) {
	issues, err := ParseIssues([]byte(`[{"number":3,"title":"Bug","body":"details","state":"open","html_url":"https://github/issue/3","user":{"login":"a"},"labels":[{"name":"bug"}]},{"number":4,"title":"PR","pull_request":{"url":"x"}}]`))
	if err != nil || len(issues) != 1 || issues[0].Number != 3 || issues[0].Labels[0] != "bug" {
		t.Fatalf("issues = %#v, err=%v", issues, err)
	}
	if err := (IssueCreateRequest{}).Validate(); err == nil {
		t.Fatal("empty issue request was accepted")
	}
	tooMany := make([]byte, 0, MaxIssues*24)
	tooMany = append(tooMany, '[')
	for i := 0; i < MaxIssues+1; i++ {
		if i > 0 {
			tooMany = append(tooMany, ',')
		}
		tooMany = append(tooMany, `{"number":1,"title":"x"}`...)
	}
	tooMany = append(tooMany, ']')
	if _, err := ParseIssues(tooMany); err == nil {
		t.Fatal("oversized issue page was accepted")
	}
}
