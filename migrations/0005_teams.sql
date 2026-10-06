-- +goose Up
CREATE TABLE teams (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    task_id TEXT NOT NULL,
    captain_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, task_id) REFERENCES tasks(tenant_id, id),
    FOREIGN KEY (tenant_id, captain_id) REFERENCES users(tenant_id, id)
);
CREATE TABLE team_members (
    tenant_id TEXT NOT NULL,
    team_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    reward_share INTEGER NOT NULL CHECK (reward_share >= 0 AND reward_share <= 100),
    consented_at TEXT,
    PRIMARY KEY (tenant_id, team_id, user_id),
    FOREIGN KEY (tenant_id, team_id) REFERENCES teams(tenant_id, id),
    FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id)
);
