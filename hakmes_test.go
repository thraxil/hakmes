package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeHandler(t *testing.T) {
	s := &site{}
	called := false
	fn := func(w http.ResponseWriter, r *http.Request, s2 *site) {
		called = true
		if s2 != s {
			t.Errorf("Expected site %v, got %v", s, s2)
		}
	}

	handler := makeHandler(fn, s)
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Expected handler function to be called")
	}
}

func TestGetMux(t *testing.T) {
	s := &site{}
	mux := getMux(s)
	if mux == nil {
		t.Fatal("getMux returned nil")
	}
}
