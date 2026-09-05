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
		if _, ok := r.URL.Query()["workspaceId"]; ok {
			t.Fatal("expected workspaceId query parameter to be omitted")
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

func TestWebhookClientCRUD(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.URL.Query().Get("workspaceId"); got != "ws_123" {
			t.Fatalf("expected workspaceId=ws_123 query param, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-key" {
			t.Fatalf("expected bearer authorization, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected JSON content type, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/webhooks":
			var body WebhookRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decoding create body: %v", err)
			}
			if body.Name != "Terraform webhook" || len(body.Triggers) != 1 || body.Triggers[0] != "link.created" {
				t.Fatalf("unexpected create body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(Webhook{ID: "wh_123", Name: body.Name, URL: body.URL, Secret: "whsec_123", Triggers: body.Triggers})
		case r.Method == http.MethodGet && r.URL.Path == "/webhooks/wh_123":
			_ = json.NewEncoder(w).Encode(Webhook{ID: "wh_123", Name: "Terraform webhook", URL: "https://example.com", Secret: "whsec_123", Triggers: []string{"link.created"}})
		case r.Method == http.MethodPatch && r.URL.Path == "/webhooks/wh_123":
			_ = json.NewEncoder(w).Encode(Webhook{ID: "wh_123", Name: "Updated webhook", URL: "https://example.com", Secret: "whsec_123", Triggers: []string{"link.updated"}})
		case r.Method == http.MethodDelete && r.URL.Path == "/webhooks/wh_123":
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "wh_123"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewWebhookClientWithBaseURL(srv.URL, "test-api-key", "ws_123", srv.Client())
	input := WebhookRequest{Name: "Terraform webhook", URL: "https://example.com", Triggers: []string{"link.created"}}
	webhook, err := c.CreateWebhook(context.Background(), input)
	if err != nil || webhook.ID != "wh_123" || webhook.Secret != "whsec_123" {
		t.Fatalf("unexpected create result: webhook=%#v err=%v", webhook, err)
	}
	if _, err := c.GetWebhook(context.Background(), "wh_123"); err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if _, err := c.UpdateWebhook(context.Background(), "wh_123", WebhookRequest{Name: "Updated webhook", URL: "https://example.com", Triggers: []string{"link.updated"}}); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if err := c.DeleteWebhook(context.Background(), "wh_123"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if calls != 4 {
		t.Fatalf("expected 4 requests, got %d", calls)
	}
}
