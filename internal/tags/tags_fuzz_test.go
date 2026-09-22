package tags

import "testing"

func FuzzParseTagRefsNeverPanics(f *testing.F) {
	f.Add([]byte("v1\x001111\x00commit\x00\x00\x00\x00\x00\x00message\x00"))
	f.Add([]byte("malformed\x00record\x00"))
	f.Fuzz(func(t *testing.T, input []byte) {
		tags, _, err := parseTagRefs(input, 64)
		if err != nil {
			return
		}
		for _, tag := range tags {
			if tag.Name == "" || tag.ObjectID == "" || tag.TargetID == "" {
				t.Fatalf("parser returned incomplete tag: %#v", tag)
			}
		}
	})
}
