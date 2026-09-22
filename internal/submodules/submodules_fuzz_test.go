package submodules

import "testing"

func FuzzParseConfigNeverPanics(f *testing.F) {
	f.Add([]byte("submodule.alpha.path\nchild\x00submodule.alpha.url\nhttps://example.test/repo\x00"))
	f.Add([]byte("malformed"))
	f.Fuzz(func(t *testing.T, input []byte) {
		modules, err := parseConfig(input)
		if err != nil {
			return
		}
		for _, module := range modules {
			if module.Path == "" {
				t.Fatal("parser returned module without a path")
			}
		}
	})
}

func FuzzParseStatusNeverPanics(f *testing.F) {
	f.Add([]byte(" 0123456789012345678901234567890123456789 child\n"))
	f.Add([]byte("malformed"))
	f.Fuzz(func(t *testing.T, input []byte) {
		status, err := parseStatus(input, 8)
		if err != nil {
			return
		}
		for path := range status {
			if path == "" {
				t.Fatal("parser returned empty submodule path")
			}
		}
	})
}
