package document

import "testing"

func FuzzParse(f *testing.F) {
	f.Add([]byte("# gitwatch:begin template=root/Go version=1\n"))
	f.Add([]byte("one\r\ntwo\n"))
	f.Fuzz(func(_ *testing.T, input []byte) { _, _ = Parse(input) })
}
