package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(srv.URL, "test-api-key", "", srv.Client())
	return c, srv
}

func TestCreateDomain(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/domains" {
			t.Fatalf("expected path /domains, got %s", r.URL.Path)
		}
		wantAuth := "Bearer " + "test-api-key"
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Fatalf("expected Authorization header, got %q", got)
		}

		var body CreateDomainRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body.Slug != "acme.com" {
			t.Fatalf("expected slug acme.com, got %s", body.Slug)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Domain{
			ID:   "dom_123",
			Slug: body.Slug,
		})
	})

	domain, err := c.CreateDomain(context.Background(), CreateDomainRequest{Slug: "acme.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain.ID != "dom_123" {
		t.Fatalf("expected id dom_123, got %s", domain.ID)
	}
	if domain.Slug != "acme.com" {
		t.Fatalf("expected slug acme.com, got %s", domain.Slug)
	}
}

func TestGetDomainNotFound(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "not_found",
				"message": "Domain not found",
			},
		})
	})

	_, err := c.GetDomain(context.Background(), "missing.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected IsNotFound to be true, got error: %v", err)
	}
}

func TestUpdateDomain(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/domains/acme.com" {
			t.Fatalf("expected path /domains/acme.com, got %s", r.URL.Path)
		}

		var body UpdateDomainRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body.Archived == nil || !*body.Archived {
			t.Fatalf("expected archived=true in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Domain{
			ID:       "dom_123",
			Slug:     "acme.com",
			Archived: true,
		})
	})

	archived := true
	domain, err := c.UpdateDomain(context.Background(), "acme.com", UpdateDomainRequest{Archived: &archived})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !domain.Archived {
		t.Fatalf("expected archived domain")
	}
}

func TestDeleteDomain(t *testing.T) {
	called := false
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/domains/acme.com" {
			t.Fatalf("expected path /domains/acme.com, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	if err := c.DeleteDomain(context.Background(), "acme.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected delete handler to be called")
	}
}

func TestListDomains(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domains" {
			t.Fatalf("expected path /domains, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("search"); got != "acme" {
			t.Fatalf("expected search=acme query param, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Domain{
			{ID: "dom_1", Slug: "acme.com"},
			{ID: "dom_2", Slug: "acme.link"},
		})
	})

	domains, err := c.ListDomains(context.Background(), ListDomainsParams{Search: "acme"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
}

func TestListDomainsIncludesWorkspaceID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("workspaceId"); got != "ws_123" {
			t.Fatalf("expected workspaceId=ws_123 query param, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Domain{})
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "test-api-key", "ws_123", srv.Client())
	if _, err := c.ListDomains(context.Background(), ListDomainsParams{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
