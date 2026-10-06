-- +goose Up
ALTER TABLE products ADD COLUMN category TEXT NOT NULL DEFAULT 'UNCATEGORIZED'
    CHECK (category IN ('COURSE', 'CLOTHING', 'CAMPUS_FOOD', 'UNCATEGORIZED'));
CREATE INDEX products_tenant_category_active_idx ON products(tenant_id, category, active);
