package managed

import "testing"

func FuzzParseManagedBlock(f *testing.F) {
	f.Add([]byte("# >>> gitwatch:gitignore begin format=1 id=root/Go source=x commit=y hash=z\n# <<< gitwatch:gitignore end format=1 id=root/Go\n"))
	f.Fuzz(func(_ *testing.T, input []byte) { _, _ = ParseManagedBlock(input) })
}
