package nativemessaging

import (
	"bytes"
	"testing"
)

func TestManifestNilAllowedOriginsIsEmptyArray(t *testing.T) {
	h := Host{Name: "org.example.app", Path: "/opt/app", AllowedExtensions: []string{"app@example.org"}}
	got, err := h.Manifest(false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte(`"allowed_origins": null`)) {
		t.Fatalf("Manifest(false) with nil AllowedOrigins wrote null: %s", got)
	}
	if !bytes.Contains(got, []byte(`"allowed_origins":[]`)) && !bytes.Contains(got, []byte(`"allowed_origins": []`)) {
		t.Fatalf("Manifest(false) missing empty allowed_origins array: %s", got)
	}
}
