package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

const (
	DatabaseEndpoint = ``
)

func TestInfo(t *testing.T) {
	db, err := DbConn(DatabaseEndpoint)
	if err != nil {
		t.Fatal(err)
	}

	h := recoveryHandler(true, infoHandler(db))

	// TODO: More test, or use assert package
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected status %d", w.Code)
	}

	// TODO: Compare to DB Query (or model)
	t.Log(w.Body.String())
}

func TestPosts(t *testing.T) {
	db, err := DbConn(DatabaseEndpoint)
	if err != nil {
		t.Fatal(err)
	}

	h := recoveryHandler(true, postsHandler(db))

	// TODO: More automatic test, or use assert package
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected status %d", w.Code)
	}

	t.Log(w.Body.String())
}

func TestImage(t *testing.T) {
	db, err := DbConn(DatabaseEndpoint)
	if err != nil {
		t.Fatal(err)
	}

	r := mux.NewRouter()
	r.Handle("/{id}", recoveryHandler(true, imageHandler("id", db)))

	// TODO: More automatic test, or use assert package
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected status %d", w.Code)
	}

	t.Log(w.Header().Get("CONTENT-TYPE"))
}

// TODO: TestUpload
