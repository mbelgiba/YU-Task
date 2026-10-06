-- +goose Up
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    org_unit_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    assignee_id TEXT,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    difficulty TEXT NOT NULL CHECK (difficulty IN ('EASY','MEDIUM','HARD')),
    reward_ec INTEGER NOT NULL CHECK (reward_ec >= 0),
    status TEXT NOT NULL CHECK (status IN ('DRAFT','OPEN','TAKEN','IN_REVIEW','ACCEPTED','REJECTED','RETURNED','DISPUTED','CANCELLED','EXPIRED')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, org_unit_id) REFERENCES org_units(tenant_id, id),
    FOREIGN KEY (tenant_id, author_id) REFERENCES users(tenant_id, id),
    FOREIGN KEY (tenant_id, assignee_id) REFERENCES users(tenant_id, id)
);
CREATE INDEX tasks_tenant_status_created_idx ON tasks(tenant_id, status, created_at);
