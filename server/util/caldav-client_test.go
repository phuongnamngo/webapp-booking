package util

import "testing"

func TestCalDAVClient_resolveHref(t *testing.T) {
	c := &CalDAVClient{url: "https://apidata.googleusercontent.com/caldav/v2"}

	abs, err := c.resolveHref("/caldav/v2/user@example.com/events/primary/abc.ics")
	if err != nil {
		t.Fatal(err)
	}
	wantAbs := "https://apidata.googleusercontent.com/caldav/v2/user@example.com/events/primary/abc.ics"
	if abs != wantAbs {
		t.Fatalf("absolute path: got %q want %q", abs, wantAbs)
	}

	rel, err := c.resolveHref("user@example.com/events/primary/abc.ics")
	if err != nil {
		t.Fatal(err)
	}
	wantRel := "https://apidata.googleusercontent.com/caldav/v2/user@example.com/events/primary/abc.ics"
	if rel != wantRel {
		t.Fatalf("relative path: got %q want %q", rel, wantRel)
	}
}
