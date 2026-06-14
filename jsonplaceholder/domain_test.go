package jsonplaceholder

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring. The client's HTTP behaviour is covered in jsonplaceholder_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "jsonplaceholder" {
		t.Errorf("Scheme = %q, want jsonplaceholder", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "jsonplaceholder" {
		t.Errorf("Identity.Binary = %q, want jsonplaceholder", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"posts/1", "post", "posts/1"},
		{"/users/2/", "post", "users/2"},
		{"https://" + Host + "/todos/3", "post", "todos/3"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("post", "1")
	want := "https://" + Host + "/posts/1"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	got, err := h.ResolveOn("jsonplaceholder", "posts/1")
	if err != nil || got.String() != "jsonplaceholder://post/posts/1" {
		t.Errorf("ResolveOn = (%q, %v), want jsonplaceholder://post/posts/1", got.String(), err)
	}
}
