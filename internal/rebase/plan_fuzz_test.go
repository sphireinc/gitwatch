package rebase

import "testing"

func FuzzParseNeverPanicsOrExceedsInputBound(f *testing.F) {
	for _, seed := range []string{
		"pick abc123 subject\n",
		"# comment\n\nreword deadbeef message\n",
		"exec echo ignored\nunknown directive\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		plan, err := Parse(input)
		if err != nil {
			return
		}
		if got := plan.Render(); len(got) > maxPlanBytes {
			t.Fatalf("rendered plan exceeded input bound: %d", len(got))
		}
	})
}
