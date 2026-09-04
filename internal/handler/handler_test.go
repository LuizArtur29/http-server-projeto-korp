package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProjetoKorp(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/projeto-korp", nil)
	rec := httptest.NewRecorder()

	before := time.Now().UTC()

	ProjetoKorp(rec, req)

	after := time.Now().UTC()

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var response ProjetoResponse

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Nome != "Projeto Korp" {
		t.Errorf(`expected nome "Projeto Korp", got %q`, response.Nome)
	}

	horario, err := time.Parse(time.RFC3339, response.Horario)
	if err != nil {
		t.Fatalf("horario is not valid RFC3339: %v", err)
	}

	_, offset := horario.Zone()
	if offset != 0 {
		t.Errorf("expected UTC timezone, got offset %d", offset)
	}

	if horario.Before(before.Add(-time.Second)) || horario.After(after.Add(time.Second)) {
		t.Errorf("returned horario %v is outside expected request interval", horario)
	}
}

func TestProjetoKorpRejectsNonGETMethods(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/projeto-korp", nil)
	rec := httptest.NewRecorder()

	ProjetoKorp(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rec.Code,
		)
	}

	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("expected Allow header GET, got %q", allow)
	}
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var response map[string]string

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf(`expected status "healthy", got %q`, response["status"])
	}
}

func TestHealthRejectsNonGETMethods(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	Health(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rec.Code,
		)
	}
}
