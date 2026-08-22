package layout

import "testing"

func TestResponsiveModes(t *testing.T) {
	if Compute(200, 60).Mode != Wide || Compute(120, 40).Mode != Medium || Compute(80, 24).Mode != Narrow || Compute(40, 10).Mode != TooSmall {
		t.Fatal("responsive mode calculation failed")
	}
	l := Compute(200, 60)
	if l.Files.Width+l.Details.Width != 200 || !l.Files.Contains(1, 3) {
		t.Fatal(l)
	}
	custom := ComputeWithSplit(200, 60, Split{FilesPercent: 50, DetailsPercent: 50})
	if custom.Files.Width != 100 || custom.Details.Width != 100 {
		t.Fatalf("custom split = %#v", custom)
	}
	invalid := ComputeWithSplit(200, 60, Split{FilesPercent: 90, DetailsPercent: 90})
	if invalid.Files.Width != 120 || invalid.Details.Width != 80 {
		t.Fatalf("invalid split did not use defaults = %#v", invalid)
	}
}
