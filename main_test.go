package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"yu-tasks/internal/config"
	"yu-tasks/internal/identity"
	"yu-tasks/internal/platform/db"
)

func TestHealthAndEmbeddedPage(t *testing.T) {
	webFS, err := fs.Sub(assets, "web")
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "routes.db"), assets)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	handler := routes(webFS, config.Config{}, store.DB, identity.NewSessions())

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

func TestDevTaskFlowEnforcesRolesRulesAndCSRF(t *testing.T) {
	store, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "tasks.db"), assets)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	webFS, err := fs.Sub(assets, "web")
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(webFS, config.Config{DevLogin: true}, store.DB, identity.NewSessions())
	staffCookie, staffCSRF := loginAs(t, handler, "STAFF")

	badBody := `{"title":"Prepare exam answers","description":"Write answers for the upcoming assessment session for a class","category":"Экзамены","difficulty":"MEDIUM","rewardEc":20}`
	blocked := apiCall(handler, http.MethodPost, "/api/tasks", badBody, staffCookie, staffCSRF)
	if blocked.Code != http.StatusUnprocessableEntity {
		t.Fatalf("prohibited task response = %d, want 422: %s", blocked.Code, blocked.Body.String())
	}

	goodBody := `{"title":"Organize archive labels","description":"Sort the department archive and label the folders by year.","category":"Office support","difficulty":"EASY","rewardEc":20}`
	withoutCSRF := apiCall(handler, http.MethodPost, "/api/tasks", goodBody, staffCookie, "")
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF response = %d, want 403", withoutCSRF.Code)
	}
	published := apiCall(handler, http.MethodPost, "/api/tasks", goodBody, staffCookie, staffCSRF)
	if published.Code != http.StatusCreated {
		t.Fatalf("publish response = %d: %s", published.Code, published.Body.String())
	}
	var created taskResponse
	if err := json.Unmarshal(published.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	staffTake := apiCall(handler, http.MethodPost, "/api/tasks/"+created.ID+"/take", `{}`, staffCookie, staffCSRF)
	if staffTake.Code != http.StatusForbidden {
		t.Fatalf("staff take response = %d, want 403", staffTake.Code)
	}

	studentCookie, studentCSRF := loginAs(t, handler, "STUDENT")
	studentPublish := apiCall(handler, http.MethodPost, "/api/tasks", goodBody, studentCookie, studentCSRF)
	if studentPublish.Code != http.StatusForbidden {
		t.Fatalf("student publish response = %d, want 403", studentPublish.Code)
	}
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			statuses <- apiCall(handler, http.MethodPost, "/api/tasks/"+created.ID+"/take", `{}`, studentCookie, studentCSRF).Code
		}()
	}
	wait.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent take statuses = %v, want one 200 and one 409", counts)
	}
	myTasks := apiCall(handler, http.MethodGet, "/api/my/tasks", "", studentCookie, "")
	if myTasks.Code != http.StatusOK || !strings.Contains(myTasks.Body.String(), created.ID) {
		t.Fatalf("my tasks response = %d: %s", myTasks.Code, myTasks.Body.String())
	}
	var audited int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE entity_id = ?`, created.ID).Scan(&audited); err != nil {
		t.Fatal(err)
	}
	if audited != 2 {
		t.Fatalf("task audit events = %d, want 2", audited)
	}
	for _, statement := range []string{
		`INSERT INTO tenants (id, name) VALUES ('tenant-other', 'Other tenant')`,
		`INSERT INTO org_units (id, tenant_id, name, kind) VALUES ('building-other', 'tenant-other', 'Other building', 'BUILDING')`,
		`INSERT INTO users (id, tenant_id, email, display_name, role) VALUES ('user-other', 'tenant-other', 'other@local.invalid', 'Other user', 'STUDENT')`,
		`INSERT INTO tasks (id, tenant_id, org_unit_id, author_id, title, description, category, difficulty, reward_ec, status) VALUES ('task-other', 'tenant-other', 'building-other', 'user-other', 'Other tenant task', 'This task belongs to another tenant.', 'Office support', 'EASY', 10, 'OPEN')`,
	} {
		if _, err := store.DB.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	otherTask := apiCall(handler, http.MethodGet, "/api/tasks", "", studentCookie, "")
	if otherTask.Code != http.StatusOK || strings.Contains(otherTask.Body.String(), "task-other") {
		t.Fatalf("task list crossed tenant boundary: %d %s", otherTask.Code, otherTask.Body.String())
	}
	otherTake := apiCall(handler, http.MethodPost, "/api/tasks/task-other/take", `{}`, studentCookie, studentCSRF)
	if otherTake.Code != http.StatusConflict {
		t.Fatalf("cross-tenant task mutation response = %d, want 409", otherTake.Code)
	}
}

func TestDeanCanConfigureProhibitedCategories(t *testing.T) {
	store, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "rules.db"), assets)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	webFS, err := fs.Sub(assets, "web")
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(webFS, config.Config{DevLogin: true}, store.DB, identity.NewSessions())
	deanCookie, deanCSRF := loginAs(t, handler, "DEAN_OFFICE")
	added := apiCall(handler, http.MethodPost, "/api/rules/prohibited", `{"category":"roof access without permit","reason":"Work at height needs authorization."}`, deanCookie, deanCSRF)
	if added.Code != http.StatusCreated {
		t.Fatalf("add rule response = %d: %s", added.Code, added.Body.String())
	}
	var rule struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(added.Body.Bytes(), &rule); err != nil {
		t.Fatal(err)
	}
	staffCookie, staffCSRF := loginAs(t, handler, "STAFF")
	blocked := apiCall(handler, http.MethodPost, "/api/tasks", `{"title":"Inspect roof equipment","description":"Inspect the roof equipment and note any visible damage.","category":"Roof access without permit","difficulty":"MEDIUM","rewardEc":15}`, staffCookie, staffCSRF)
	if blocked.Code != http.StatusUnprocessableEntity {
		t.Fatalf("configured category check = %d, want 422", blocked.Code)
	}
	studentCookie, studentCSRF := loginAs(t, handler, "STUDENT")
	forbidden := apiCall(handler, http.MethodPost, "/api/rules/prohibited", `{"category":"test category","reason":"test reason"}`, studentCookie, studentCSRF)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("student rule edit = %d, want 403", forbidden.Code)
	}
	disabled := apiCall(handler, http.MethodPost, "/api/rules/prohibited/"+rule.ID+"/disable", `{}`, deanCookie, deanCSRF)
	if disabled.Code != http.StatusOK {
		t.Fatalf("disable rule = %d: %s", disabled.Code, disabled.Body.String())
	}
	reopened := apiCall(handler, http.MethodPost, "/api/tasks", `{"title":"Inspect roof equipment","description":"Inspect the roof equipment and note any visible damage.","category":"Roof access without permit","difficulty":"MEDIUM","rewardEc":15}`, staffCookie, staffCSRF)
	if reopened.Code != http.StatusCreated {
		t.Fatalf("disabled category should allow publication, got %d: %s", reopened.Code, reopened.Body.String())
	}
}

func TestShopCatalogUsesStoredProductsAndStaffPermissions(t *testing.T) {
	store, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "shop.db"), assets)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	webFS, err := fs.Sub(assets, "web")
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(webFS, config.Config{DevLogin: true}, store.DB, identity.NewSessions())
	staffCookie, staffCSRF := loginAs(t, handler, "STAFF")
	productJSON := `{"name":"Вымышленная кружка","description":"Эта запись создана тестом и не появится в каталоге пилотной базы.","priceEc":25,"stock":3}`
	created := apiCall(handler, http.MethodPost, "/api/products", productJSON, staffCookie, staffCSRF)
	if created.Code != http.StatusCreated {
		t.Fatalf("staff add product = %d: %s", created.Code, created.Body.String())
	}
	studentCookie, _ := loginAs(t, handler, "STUDENT")
	listed := apiCall(handler, http.MethodGet, "/api/products", "", studentCookie, "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), "Вымышленная кружка") {
		t.Fatalf("stored product not shown to student: %d %s", listed.Code, listed.Body.String())
	}
	denied := apiCall(handler, http.MethodPost, "/api/products", productJSON, studentCookie, "test-token")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("student add product = %d, want 403", denied.Code)
	}
}

func loginAs(t *testing.T, handler http.Handler, role string) (*http.Cookie, string) {
	t.Helper()
	response := apiCall(handler, http.MethodPost, "/api/dev-login", `{"role":"`+role+`"}`, nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("dev-login %s response = %d: %s", role, response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("dev-login did not set a session cookie")
	}
	var session struct {
		CSRF string `json:"csrfToken"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	return cookies[0], session.CSRF
}

func apiCall(handler http.Handler, method, path, body string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "http://127.0.0.1"+path, strings.NewReader(body))
	request.Header.Set("Origin", "http://127.0.0.1")
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
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
