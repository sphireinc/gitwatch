package plugin

import (
	"bytes"
	"testing"
)

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

func TestDecodeRejectsHostileMessageSizes(t *testing.T) {
	if _, err := Decode([]byte("{\"type\":\"status\"}\n{\"type\":\"extra\"}")); err == nil {
		t.Fatal("multiple JSON messages were accepted")
	}
	if _, err := Decode([]byte(`{"type":"` + string(make([]byte, MaxFieldBytes+1)) + `"}`)); err == nil {
		t.Fatal("oversized type field was accepted")
	}
	if _, err := Decode(bytes.Repeat([]byte{'x'}, MaxMessageBytes+1)); err == nil {
		t.Fatal("oversized message was accepted")
	}
}
