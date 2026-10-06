package main

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"yu-tasks/internal/config"
	"yu-tasks/internal/platform/db"
)

func TestHealthAndEmbeddedPage(t *testing.T) {
	webFS, err := fs.Sub(assets, "web")
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(webFS, config.Config{})

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK || strings.TrimSpace(health.Body.String()) != "ok" {
		t.Fatalf("health response = %d %q", health.Code, health.Body.String())
	}

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "YU Tasks") {
		t.Fatalf("embedded page response = %d", page.Code)
	}
	if page.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("expected Content-Security-Policy header")
	}
}

func TestEmbeddedMigrationsCreateImmutableLedger(t *testing.T) {
	store, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"), assets)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := store.DB.Exec(`INSERT INTO tenants (id, name) VALUES ('tenant-a', 'Test')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO ledger_operations (id, tenant_id, idempotency_key, kind) VALUES ('op-a', 'tenant-a', 'once', 'ISSUE')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO ledger_entries (id, tenant_id, operation_id, account_id, amount_ec) VALUES ('entry-a', 'tenant-a', 'op-a', 'account-a', 10)`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`INSERT INTO ledger_entries (id, tenant_id, operation_id, account_id, amount_ec) VALUES ('entry-b', 'tenant-a', 'op-a', 'university-ec', -10)`); err != nil {
		t.Fatal(err)
	}
	var total int64
	if err := store.DB.QueryRow(`SELECT COALESCE(SUM(amount_ec), 0) FROM ledger_entries WHERE tenant_id = 'tenant-a' AND operation_id = 'op-a'`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("ledger operation is unbalanced: sum = %d", total)
	}
	if _, err := store.DB.Exec(`UPDATE ledger_entries SET amount_ec = 20 WHERE id = 'entry-a'`); err == nil {
		t.Fatal("expected ledger update trigger to reject mutation")
	}
	if _, err := store.DB.Exec(`DELETE FROM ledger_entries WHERE id = 'entry-a'`); err == nil {
		t.Fatal("expected ledger delete trigger to reject mutation")
	}
}
