package util

import "testing"

func TestGoogleCalDAVPrincipalURL(t *testing.T) {
	url := GoogleCalDAVPrincipalURL("user@example.com")
	want := GoogleCalDAVRootURL
	if url != want {
		t.Fatalf("got %q want %q", url, want)
	}
}
