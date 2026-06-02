package xui

import "testing"

func TestJoinPathPreservesBasePath(t *testing.T) {
	c, err := NewClient("http://localhost:55002/tf-acc/", "u", "p", true)
	if err != nil {
		t.Fatal(err)
	}
	csrf, err := c.join("csrf-token")
	if err != nil {
		t.Fatal(err)
	}
	if csrf != "http://localhost:55002/tf-acc/csrf-token" {
		t.Fatalf("csrf URL = %q", csrf)
	}
	login, err := c.join("login")
	if err != nil {
		t.Fatal(err)
	}
	if login != "http://localhost:55002/tf-acc/login" {
		t.Fatalf("login URL = %q", login)
	}
}
