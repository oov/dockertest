package dockertest

import "testing"

func TestEvalDockerHost(t *testing.T) {
	if g, w := evalDockerHost(""), "127.0.0.1"; g != w {
		t.Errorf("want %q got %q", w, g)
	}
	if g, w := evalDockerHost("tcp://1.2.3.4"), "1.2.3.4"; g != w {
		t.Errorf("want %q got %q", w, g)
	}
	if g, w := evalDockerHost("tcp://1.2.3.4:5678"), "1.2.3.4"; g != w {
		t.Errorf("want %q got %q", w, g)
	}
}

func TestSuggestMappingHost(t *testing.T) {
	if g, w := suggestMappingHost(""), "127.0.0.1"; g != w {
		t.Errorf("want %q got %q", w, g)
	}
	if g, w := suggestMappingHost("tcp://1.2.3.4"), "0.0.0.0"; g != w {
		t.Errorf("want %q got %q", w, g)
	}
	if g, w := suggestMappingHost("tcp://1.2.3.4:5678"), "0.0.0.0"; g != w {
		t.Errorf("want %q got %q", w, g)
	}
}
