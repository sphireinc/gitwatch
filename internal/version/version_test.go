package version

import "testing"

func TestResolvedVersion(t *testing.T) {
	tests := []struct {
		name   string
		linked string
		module string
		want   string
	}{
		{name: "release linker value wins", linked: "1.0.7", module: "v1.0.7", want: "1.0.7"},
		{name: "tagged source install", linked: "1.0.0-dev", module: "v1.0.7", want: "1.0.7"},
		{name: "tagged prerelease source install", linked: "1.0.0-dev", module: "v1.0.8-rc.1", want: "1.0.8-rc.1"},
		{name: "development build", linked: "1.0.0-dev", module: "(devel)", want: "1.0.0-dev"},
		{name: "dirty checkout", linked: "1.0.0-dev", module: "v1.0.7+dirty", want: "1.0.0-dev"},
		{name: "pseudo version is development", linked: "1.0.0-dev", module: "v0.0.0-20260915120000-abcdefabcdef", want: "1.0.0-dev"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolvedVersion(test.linked, test.module); got != test.want {
				t.Fatalf("resolvedVersion(%q, %q) = %q, want %q", test.linked, test.module, got, test.want)
			}
		})
	}
}
