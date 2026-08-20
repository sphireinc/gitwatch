package plugin

import "testing"

func TestWireCompatibility(t *testing.T) {
	want := Message{Type: "status", ID: "1", Payload: []byte(`{"text":"ready"}`)}
	data, err := Encode(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil || got.Type != want.Type || string(got.Payload) != string(want.Payload) {
		t.Fatalf("unexpected message: %#v, %v", got, err)
	}
}
