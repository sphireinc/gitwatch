package customcmd

import "testing"

func FuzzDefinitionExpandNeverProducesUnsafeArg(f *testing.F) {
	for _, seed := range []string{
		"{path}",
		"--branch={branch}",
		"plain value",
		"{unknown}",
		"line\nfeed",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		definition := Definition{Name: "fuzz", Executable: "tool", Contexts: []string{"any"}, Args: []string{value}}
		invocation, err := definition.Expand(Context{
			RepositoryRoot: "/repo",
			SelectedPath:   "path with spaces",
			Branch:         "main",
		})
		if err != nil {
			return
		}
		for _, arg := range invocation.Args {
			for _, forbidden := range []byte{'\x00', '\r', '\n'} {
				for index := 0; index < len(arg); index++ {
					if arg[index] == forbidden {
						t.Fatalf("expanded argv contains forbidden byte %q", forbidden)
					}
				}
			}
		}
	})
}
