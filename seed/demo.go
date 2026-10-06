// Package seed contains data used only by the local development preview.
package seed

import (
	"context"
	"database/sql"
	"fmt"

	"yu-tasks/internal/ledger"
)

type product struct {
	id, category, name, description string
	price, stock                    int64
}

var demoProducts = []product{
	{"demo-course-python", "COURSE", "Вводный курс Python / Intro to Python", "Учебный макет: основы программирования. Demo listing; course details are illustrative.", 2500, 20},
	{"demo-clothing-hoodie", "CLOTHING", "Худи YU Tasks / YU Tasks hoodie", "Макет университетского мерча. Demo mockup; not an official university product.", 4500, 12},
	{"demo-food-lunch", "CAMPUS_FOOD", "Обеденный набор / Campus lunch set", "Демонстрационная позиция еды в кампусе. Demo listing; venue and menu are not confirmed.", 1200, 30},
}

// EnsureDevelopmentDemo is intentionally restricted to the local pilot tenant.
// Its catalog values, quantities, prices and welcome EC grants are fictional.
func EnsureDevelopmentDemo(ctx context.Context, database *sql.DB, tenantID string) error {
	if tenantID != "pilot-yessenov" {
		return fmt.Errorf("development demo seed is disabled for tenant %q", tenantID)
	}
	for _, item := range demoProducts {
		if _, err := database.ExecContext(ctx, `INSERT INTO products (id, tenant_id, category, name, description, price_ec, stock) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT (tenant_id, id) DO NOTHING`, item.id, tenantID, item.category, item.name, item.description, item.price, item.stock); err != nil {
			return fmt.Errorf("seed demo catalog item %s: %w", item.id, err)
		}
	}
	rows, err := database.QueryContext(ctx, `SELECT id FROM users WHERE tenant_id = ? ORDER BY id`, tenantID)
	if err != nil {
		return fmt.Errorf("list demo accounts: %w", err)
	}
	userIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			return fmt.Errorf("read demo account: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read demo accounts: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close demo accounts: %w", err)
	}
	service := ledger.New(database)
	for _, userID := range userIDs {
		if err := service.EnsureDemoWelcomeBalance(ctx, tenantID, userID); err != nil {
			return fmt.Errorf("grant demo EC to %s: %w", userID, err)
		}
	}
	return nil
}
