package match

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
)

func FuzzMatch(f *testing.F) {
	f.Add([]byte("*.log\n# comment\n"))
	f.Fuzz(func(t *testing.T, input []byte) {
		doc, err := document.Parse(input)
		if err != nil {
			return
		}
		cat, err := catalog.Default()
		if err != nil {
			t.Fatal(err)
		}
		_ = Match(doc, cat)
	})
}
