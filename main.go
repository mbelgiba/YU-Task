package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"yu-tasks/internal/config"
	"yu-tasks/internal/identity"
	"yu-tasks/internal/ledger"
	"yu-tasks/internal/platform/db"
	"yu-tasks/seed"
)

//go:embed web migrations
var assets embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := db.Open(ctx, cfg.DBPath, assets)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	webFS, err := fs.Sub(assets, "web")
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: cfg.Address, Handler: routes(webFS, cfg, store.DB, identity.NewSessions()), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Printf("YU Tasks listening on %s (%s)", cfg.Address, cfg.Environment)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

type taskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Difficulty  string `json:"difficulty"`
	RewardEC    int64  `json:"rewardEc"`
}

type taskResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Difficulty  string `json:"difficulty"`
	RewardEC    int64  `json:"rewardEc"`
	Status      string `json:"status"`
}

type productInput struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceEC     int64  `json:"priceEc"`
	Stock       int64  `json:"stock"`
}

type productResponse struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceEC     int64  `json:"priceEc"`
	Stock       int64  `json:"stock"`
	Demo        bool   `json:"demo"`
}

func routes(webFS fs.FS, cfg config.Config, database *sql.DB, sessions *identity.Sessions) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, securityHeaders)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	r.Get("/api/config", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"devLogin": cfg.DevLogin, "languages": []string{"ru", "en"}})
	})
	r.Get("/api/session", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "userId": session.UserID, "role": session.Role, "csrfToken": session.CSRF})
	})
	r.Get("/api/ledger/balance", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in to view your EC balance")
			return
		}
		balance, err := ledger.New(database).Balance(r.Context(), session.TenantID, session.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load EC balance")
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"balanceEc": balance})
	})
	r.Get("/api/rules/ledger", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok || !mayManageRules(session.Role) {
			writeError(w, http.StatusForbidden, "dean office access is required")
			return
		}
		var emission, student int64
		for _, setting := range []struct {
			key   string
			value *int64
		}{
			{"monthly_emission_limit_ec:pilot-building", &emission},
			{"student_monthly_earning_limit_ec", &student},
		} {
			err := database.QueryRowContext(r.Context(), `SELECT CAST(rule_value AS INTEGER) FROM tenant_rules WHERE tenant_id = ? AND rule_key = ?`, session.TenantID, setting.key).Scan(setting.value)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusInternalServerError, "could not load EC limits")
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]int64{"monthlyEmissionLimitEc": emission, "studentMonthlyEarningLimitEc": student})
	})
	r.Post("/api/rules/ledger", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok || !mayManageRules(session.Role) {
			writeError(w, http.StatusForbidden, "dean office access is required")
			return
		}
		if !validCSRF(r, session) || !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "request could not be verified")
			return
		}
		var input struct {
			MonthlyEmissionLimitEC       int64 `json:"monthlyEmissionLimitEc"`
			StudentMonthlyEarningLimitEC int64 `json:"studentMonthlyEarningLimitEc"`
		}
		if decodeJSON(w, r, &input) != nil || input.MonthlyEmissionLimitEC <= 0 || input.StudentMonthlyEarningLimitEC <= 0 {
			writeError(w, http.StatusBadRequest, "both monthly limits must be positive whole EC amounts")
			return
		}
		tx, err := database.BeginTx(r.Context(), nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not save EC limits")
			return
		}
		defer tx.Rollback()
		for _, setting := range []struct {
			key   string
			value int64
		}{
			{"monthly_emission_limit_ec:pilot-building", input.MonthlyEmissionLimitEC},
			{"student_monthly_earning_limit_ec", input.StudentMonthlyEarningLimitEC},
		} {
			if _, err = tx.ExecContext(r.Context(), `INSERT INTO tenant_rules (tenant_id, rule_key, rule_value) VALUES (?, ?, ?) ON CONFLICT (tenant_id, rule_key) DO UPDATE SET rule_value = excluded.rule_value, updated_at = CURRENT_TIMESTAMP`, session.TenantID, setting.key, fmt.Sprint(setting.value)); err != nil {
				writeError(w, http.StatusInternalServerError, "could not save EC limits")
				return
			}
		}
		if err = recordAudit(r.Context(), tx, session, "EC_LIMITS_UPDATED", "tenant_rules", "pilot-building", r.RemoteAddr); err != nil {
			writeError(w, http.StatusInternalServerError, "could not record EC policy change")
			return
		}
		if err = tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save EC limits")
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"monthlyEmissionLimitEc": input.MonthlyEmissionLimitEC, "studentMonthlyEarningLimitEc": input.StudentMonthlyEarningLimitEC})
	})
	r.Post("/api/ledger/issue", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok || (session.Role != "DEAN_OFFICE" && session.Role != "RECTOR") {
			writeError(w, http.StatusForbidden, "dean office or rector access is required")
			return
		}
		if !validCSRF(r, session) || !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "request could not be verified")
			return
		}
		var input struct {
			StudentID      string `json:"studentId"`
			AmountEC       int64  `json:"amountEc"`
			IdempotencyKey string `json:"idempotencyKey"`
		}
		if decodeJSON(w, r, &input) != nil || input.AmountEC <= 0 || len(input.StudentID) > 120 || len(input.IdempotencyKey) < 16 || len(input.IdempotencyKey) > 120 {
			writeError(w, http.StatusBadRequest, "student, positive EC amount and unique request key are required")
			return
		}
		err := ledger.New(database).Issue(r.Context(), session.TenantID, "pilot-building", session.UserID, input.StudentID, input.AmountEC, input.IdempotencyKey)
		switch {
		case errors.Is(err, ledger.ErrPolicyMissing):
			writeError(w, http.StatusConflict, "configure both EC limits before issuing bonus units")
		case errors.Is(err, ledger.ErrPolicyExceeded):
			writeError(w, http.StatusUnprocessableEntity, "configured EC limit would be exceeded")
		case errors.Is(err, ledger.ErrStudentNotFound):
			writeError(w, http.StatusNotFound, "student account was not found")
		case errors.Is(err, ledger.ErrIdempotencyConflict):
			writeError(w, http.StatusConflict, "request key has already been used")
		case err != nil:
			writeError(w, http.StatusInternalServerError, "could not issue EC bonus units")
		default:
			writeJSON(w, http.StatusCreated, map[string]int64{"issuedEc": input.AmountEC})
		}
	})
	r.Get("/api/products", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in to view the catalog")
			return
		}
		rows, err := database.QueryContext(r.Context(), `SELECT id, category, name, description, price_ec, stock FROM products WHERE tenant_id = ? AND active = 1 ORDER BY category, created_at DESC LIMIT 100`, session.TenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load catalog")
			return
		}
		defer rows.Close()
		products := make([]productResponse, 0)
		for rows.Next() {
			var product productResponse
			if err := rows.Scan(&product.ID, &product.Category, &product.Name, &product.Description, &product.PriceEC, &product.Stock); err != nil {
				writeError(w, http.StatusInternalServerError, "could not read catalog")
				return
			}
			product.Demo = strings.HasPrefix(product.ID, "demo-")
			products = append(products, product)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read catalog")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"products": products})
	})
	r.Post("/api/products", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok || !mayPublish(session.Role) {
			writeError(w, http.StatusForbidden, "staff access is required to add a product")
			return
		}
		if !validCSRF(r, session) || !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "request could not be verified")
			return
		}
		var input productInput
		if decodeJSON(w, r, &input) != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		input.Category = strings.ToUpper(strings.TrimSpace(input.Category))
		input.Name, input.Description = strings.TrimSpace(input.Name), strings.TrimSpace(input.Description)
		if input.Category != "COURSE" && input.Category != "CLOTHING" && input.Category != "CAMPUS_FOOD" {
			writeError(w, http.StatusBadRequest, "category must be COURSE, CLOTHING or CAMPUS_FOOD")
			return
		}
		if len([]rune(input.Name)) < 2 || len([]rune(input.Name)) > 120 || len([]rune(input.Description)) > 1000 || input.PriceEC <= 0 || input.Stock < 0 {
			writeError(w, http.StatusBadRequest, "check name, description, EC price and stock")
			return
		}
		id, err := newID()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not create product")
			return
		}
		tx, err := database.BeginTx(r.Context(), nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not save product")
			return
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(r.Context(), `INSERT INTO products (id, tenant_id, category, name, description, price_ec, stock) VALUES (?, ?, ?, ?, ?, ?, ?)`, id, session.TenantID, input.Category, input.Name, input.Description, input.PriceEC, input.Stock); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save product")
			return
		}
		if err = recordAudit(r.Context(), tx, session, "PRODUCT_ADDED", "product", id, r.RemoteAddr); err != nil {
			writeError(w, http.StatusInternalServerError, "could not record product change")
			return
		}
		if err = tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save product")
			return
		}
		writeJSON(w, http.StatusCreated, productResponse{ID: id, Category: input.Category, Name: input.Name, Description: input.Description, PriceEC: input.PriceEC, Stock: input.Stock})
	})
	r.Post("/api/dev-login", func(w http.ResponseWriter, r *http.Request) {
		if !cfg.DevLogin || !sameOrigin(r) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var request struct {
			Role string `json:"role"`
		}
		if decodeJSON(w, r, &request) != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		role := strings.ToUpper(strings.TrimSpace(request.Role))
		if !validRole(role) {
			writeError(w, http.StatusBadRequest, "unknown role")
			return
		}
		if err := ensureDevUser(r.Context(), database, role); err != nil {
			writeError(w, http.StatusInternalServerError, "could not initialize local account")
			return
		}
		sessionID, session, err := sessions.Create("dev-"+strings.ToLower(role), "pilot-yessenov", role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not create session")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: identity.CookieName, Value: sessionID, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(identity.MaxAge.Seconds())})
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "role": role, "csrfToken": session.CSRF})
	})
	r.Post("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok || !validCSRF(r, session) || !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "request could not be verified")
			return
		}
		sessions.Delete(r)
		http.SetCookie(w, &http.Cookie{Name: identity.CookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
	})
	r.Get("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in to view tasks")
			return
		}
		queryTasks(w, r, database, session, false)
	})
	r.Get("/api/my/tasks", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in to view your work")
			return
		}
		queryTasks(w, r, database, session, true)
	})
	r.Get("/api/rules/prohibited", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok || !mayManageRules(session.Role) {
			writeError(w, http.StatusForbidden, "dean office access is required")
			return
		}
		rows, err := database.QueryContext(r.Context(), `SELECT id, category, reason, active FROM prohibited_categories WHERE tenant_id = ? ORDER BY category`, session.TenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load prohibited categories")
			return
		}
		defer rows.Close()
		type rule struct {
			ID       string `json:"id"`
			Category string `json:"category"`
			Reason   string `json:"reason"`
			Active   bool   `json:"active"`
		}
		items := make([]rule, 0)
		for rows.Next() {
			var item rule
			var active int
			if err := rows.Scan(&item.ID, &item.Category, &item.Reason, &active); err != nil {
				writeError(w, http.StatusInternalServerError, "could not read prohibited categories")
				return
			}
			item.Active = active == 1
			items = append(items, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{"categories": items})
	})
	r.Post("/api/rules/prohibited", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok || !mayManageRules(session.Role) {
			writeError(w, http.StatusForbidden, "dean office access is required")
			return
		}
		if !validCSRF(r, session) || !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "request could not be verified")
			return
		}
		var input struct {
			Category string `json:"category"`
			Reason   string `json:"reason"`
		}
		if decodeJSON(w, r, &input) != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		input.Category, input.Reason = strings.TrimSpace(input.Category), strings.TrimSpace(input.Reason)
		if len([]rune(input.Category)) < 2 || len([]rune(input.Category)) > 100 || len([]rune(input.Reason)) < 3 || len([]rune(input.Reason)) > 300 {
			writeError(w, http.StatusBadRequest, "check category and reason lengths")
			return
		}
		id, err := newID()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not create rule")
			return
		}
		tx, err := database.BeginTx(r.Context(), nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not save rule")
			return
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(r.Context(), `INSERT INTO prohibited_categories (id, tenant_id, org_unit_id, category, reason) VALUES (?, ?, 'pilot-building', ?, ?)`, id, session.TenantID, input.Category, input.Reason); err != nil {
			writeError(w, http.StatusConflict, "category already exists or could not be saved")
			return
		}
		if err = recordAudit(r.Context(), tx, session, "PROHIBITED_CATEGORY_ADDED", "prohibited_category", id, r.RemoteAddr); err != nil {
			writeError(w, http.StatusInternalServerError, "could not record rule change")
			return
		}
		if err = tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save rule")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id, "category": input.Category, "reason": input.Reason})
	})
	r.Post("/api/rules/prohibited/{ruleID}/disable", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok || !mayManageRules(session.Role) {
			writeError(w, http.StatusForbidden, "dean office access is required")
			return
		}
		if !validCSRF(r, session) || !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "request could not be verified")
			return
		}
		tx, err := database.BeginTx(r.Context(), nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not update rule")
			return
		}
		defer tx.Rollback()
		result, err := tx.ExecContext(r.Context(), `UPDATE prohibited_categories SET active = 0 WHERE tenant_id = ? AND id = ? AND active = 1`, session.TenantID, chi.URLParam(r, "ruleID"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not update rule")
			return
		}
		changed, err := result.RowsAffected()
		if err != nil || changed == 0 {
			writeError(w, http.StatusNotFound, "active rule was not found")
			return
		}
		if err = recordAudit(r.Context(), tx, session, "PROHIBITED_CATEGORY_DISABLED", "prohibited_category", chi.URLParam(r, "ruleID"), r.RemoteAddr); err != nil {
			writeError(w, http.StatusInternalServerError, "could not record rule change")
			return
		}
		if err = tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not update rule")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"active": false})
	})
	r.Post("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in to publish a task")
			return
		}
		if !validCSRF(r, session) || !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "request could not be verified")
			return
		}
		if !mayPublish(session.Role) {
			writeError(w, http.StatusForbidden, "your role cannot publish tasks")
			return
		}
		var input taskInput
		if decodeJSON(w, r, &input) != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		input.Title = strings.TrimSpace(input.Title)
		input.Description = strings.TrimSpace(input.Description)
		input.Category = strings.TrimSpace(input.Category)
		input.Difficulty = strings.ToUpper(strings.TrimSpace(input.Difficulty))
		if len([]rune(input.Title)) < 4 || len([]rune(input.Title)) > 160 || len([]rune(input.Description)) < 10 || len([]rune(input.Description)) > 5000 || len([]rune(input.Category)) < 2 || len([]rune(input.Category)) > 100 {
			writeError(w, http.StatusBadRequest, "check title, description and category lengths")
			return
		}
		if input.Difficulty != "EASY" && input.Difficulty != "MEDIUM" && input.Difficulty != "HARD" {
			writeError(w, http.StatusBadRequest, "difficulty must be EASY, MEDIUM or HARD")
			return
		}
		if input.RewardEC <= 0 {
			writeError(w, http.StatusBadRequest, "EC reward must be positive")
			return
		}
		prohibited, err := isProhibited(r.Context(), database, session.TenantID, input.Category, input.Title+" "+input.Description)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not check task rules")
			return
		}
		if prohibited {
			writeError(w, http.StatusUnprocessableEntity, "this task matches a prohibited category configured by the dean office")
			return
		}
		id, err := newID()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not create task")
			return
		}
		tx, err := database.BeginTx(r.Context(), nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not publish task")
			return
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(r.Context(), `INSERT INTO tasks (id, tenant_id, org_unit_id, author_id, title, description, category, difficulty, reward_ec, status) VALUES (?, ?, 'pilot-building', ?, ?, ?, ?, ?, ?, 'OPEN')`, id, session.TenantID, session.UserID, input.Title, input.Description, input.Category, input.Difficulty, input.RewardEC); err != nil {
			writeError(w, http.StatusInternalServerError, "could not publish task")
			return
		}
		if err = recordAudit(r.Context(), tx, session, "TASK_PUBLISHED", "task", id, r.RemoteAddr); err != nil {
			writeError(w, http.StatusInternalServerError, "could not record task publication")
			return
		}
		if err = tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not publish task")
			return
		}
		writeJSON(w, http.StatusCreated, taskResponse{ID: id, Title: input.Title, Description: input.Description, Category: input.Category, Difficulty: input.Difficulty, RewardEC: input.RewardEC, Status: "OPEN"})
	})
	r.Post("/api/tasks/{taskID}/take", func(w http.ResponseWriter, r *http.Request) {
		session, ok := sessions.Get(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in to take a task")
			return
		}
		if !validCSRF(r, session) || !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "request could not be verified")
			return
		}
		if session.Role != "STUDENT" {
			writeError(w, http.StatusForbidden, "only students can take a task")
			return
		}
		tx, err := database.BeginTx(r.Context(), nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not take task")
			return
		}
		defer tx.Rollback()
		result, err := tx.ExecContext(r.Context(), `UPDATE tasks SET assignee_id = ?, status = 'TAKEN', updated_at = CURRENT_TIMESTAMP WHERE tenant_id = ? AND id = ? AND status = 'OPEN'`, session.UserID, session.TenantID, chi.URLParam(r, "taskID"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not take task")
			return
		}
		changed, err := result.RowsAffected()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not verify task status")
			return
		}
		if changed == 0 {
			writeError(w, http.StatusConflict, "task is no longer open")
			return
		}
		if err := recordAudit(r.Context(), tx, session, "TASK_TAKEN", "task", chi.URLParam(r, "taskID"), r.RemoteAddr); err != nil {
			writeError(w, http.StatusInternalServerError, "could not record task assignment")
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not take task")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "TAKEN"})
	})
	r.Handle("/*", http.FileServer(http.FS(webFS)))
	return r
}

func ensureDevUser(ctx context.Context, database *sql.DB, role string) error {
	const tenantID = "pilot-yessenov"
	if _, err := database.ExecContext(ctx, `INSERT INTO tenants (id, name) VALUES (?, ?) ON CONFLICT (id) DO NOTHING`, tenantID, "Yessenov University"); err != nil {
		return err
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO org_units (id, tenant_id, name, kind, building) VALUES ('pilot-building', ?, '[ВОПРОС: корпус]', 'BUILDING', '[ВОПРОС: корпус]') ON CONFLICT (tenant_id, id) DO NOTHING`, tenantID); err != nil {
		return err
	}
	userID := "dev-" + strings.ToLower(role)
	_, err := database.ExecContext(ctx, `INSERT INTO users (id, tenant_id, email, display_name, role) VALUES (?, ?, ?, ?, ?) ON CONFLICT (tenant_id, email) DO UPDATE SET display_name = excluded.display_name, role = excluded.role`, userID, tenantID, userID+"@local.invalid", "Local development account ("+role+")", role)
	if err != nil {
		return err
	}
	defaults := [][2]string{{"курсовые", "Работы, влияющие на оценивание"}, {"контрольные", "Работы, влияющие на оценивание"}, {"экзамены", "Работы, влияющие на оценивание"}, {"электромонтаж без допуска", "Опасные работы без допуска"}, {"работы на высоте без допуска", "Опасные работы без допуска"}}
	for _, rule := range defaults {
		if _, err := database.ExecContext(ctx, `INSERT INTO prohibited_categories (id, tenant_id, org_unit_id, category, reason) VALUES (?, ?, 'pilot-building', ?, ?) ON CONFLICT (tenant_id, org_unit_id, category) DO NOTHING`, "default-"+strings.ReplaceAll(rule[0], " ", "-"), tenantID, rule[0], rule[1]); err != nil {
			return err
		}
	}
	return seed.EnsureDevelopmentDemo(ctx, database, tenantID)
}

func queryTasks(w http.ResponseWriter, r *http.Request, database *sql.DB, session identity.Session, mine bool) {
	query := `SELECT id, title, description, category, difficulty, reward_ec, status FROM tasks WHERE tenant_id = ? AND status = 'OPEN' ORDER BY created_at DESC LIMIT 100`
	args := []any{session.TenantID}
	if mine {
		query = `SELECT id, title, description, category, difficulty, reward_ec, status FROM tasks WHERE tenant_id = ? AND (author_id = ? OR assignee_id = ?) ORDER BY created_at DESC LIMIT 100`
		args = append(args, session.UserID, session.UserID)
	}
	rows, err := database.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load tasks")
		return
	}
	defer rows.Close()
	tasks := make([]taskResponse, 0)
	for rows.Next() {
		var item taskResponse
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Category, &item.Difficulty, &item.RewardEC, &item.Status); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read tasks")
			return
		}
		tasks = append(tasks, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read tasks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func isProhibited(ctx context.Context, database *sql.DB, tenantID, category, text string) (bool, error) {
	rows, err := database.QueryContext(ctx, `SELECT category FROM prohibited_categories WHERE tenant_id = ? AND active = 1`, tenantID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var blockedCategory string
		if err := rows.Scan(&blockedCategory); err != nil {
			return false, err
		}
		if strings.EqualFold(category, blockedCategory) || containsPhrase(text, blockedCategory) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func containsPhrase(text, phrase string) bool {
	words := func(value string) []string {
		return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	}
	textWords, phraseWords := words(text), words(phrase)
	if len(phraseWords) == 0 || len(phraseWords) > len(textWords) {
		return false
	}
	for start := 0; start+len(phraseWords) <= len(textWords); start++ {
		match := true
		for offset, word := range phraseWords {
			if textWords[start+offset] != word {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func validCSRF(r *http.Request, session identity.Session) bool {
	provided := r.Header.Get("X-CSRF-Token")
	return session.CSRF != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(session.CSRF)) == 1
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return r.Header.Get("Sec-Fetch-Site") == "same-origin"
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host != r.Host || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return false
	}
	return (r.TLS == nil && parsed.Scheme == "http") || (r.TLS != nil && parsed.Scheme == "https")
}

func validRole(role string) bool {
	switch role {
	case "STUDENT", "STAFF", "DEAN_OFFICE", "RECTOR", "PLATFORM_OWNER":
		return true
	default:
		return false
	}
}

func mayPublish(role string) bool {
	return role == "STAFF" || role == "DEAN_OFFICE" || role == "RECTOR" || role == "PLATFORM_OWNER"
}

func mayManageRules(role string) bool {
	return role == "DEAN_OFFICE" || role == "RECTOR" || role == "PLATFORM_OWNER"
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("content type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request must contain exactly one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func newID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func recordAudit(ctx context.Context, tx *sql.Tx, session identity.Session, action, entityType, entityID, remoteAddr string) error {
	id, err := newID()
	if err != nil {
		return err
	}
	var ip any
	if host, _, splitErr := net.SplitHostPort(remoteAddr); splitErr == nil && net.ParseIP(host) != nil {
		ip = host
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events (id, tenant_id, actor_id, action, entity_type, entity_id, ip_address) VALUES (?, ?, ?, ?, ?, ?, ?)`, id, session.TenantID, session.UserID, action, entityType, entityID, ip)
	return err
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", strings.Join([]string{"default-src 'self'", "img-src 'self' data:", "style-src 'self'", "script-src 'self'", "connect-src 'self'", "object-src 'none'", "base-uri 'self'", "frame-ancestors 'none'"}, "; "))
		next.ServeHTTP(w, r)
	})
}
