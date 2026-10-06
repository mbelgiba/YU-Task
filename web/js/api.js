export async function loadAppConfig() {
  const response = await fetch("/api/config", { headers: { Accept: "application/json" } });
  if (!response.ok) throw new Error(`Configuration request failed (${response.status})`);
  return response.json();
}

async function request(path, options = {}) {
  const response = await fetch(path, {
    credentials: "same-origin",
    ...options,
    headers: { Accept: "application/json", ...options.headers },
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body.error || `Request failed (${response.status})`);
  return body;
}

export const loadSession = () => request("/api/session");

export const devLogin = (role) => request("/api/dev-login", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ role }),
});

export const logout = (csrfToken) => request("/api/logout", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
  body: "{}",
});

export const listTasks = () => request("/api/tasks");
export const listMyTasks = () => request("/api/my/tasks");

export const publishTask = (csrfToken, task) => request("/api/tasks", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
  body: JSON.stringify(task),
});

export const takeTask = (csrfToken, taskID) => request(`/api/tasks/${encodeURIComponent(taskID)}/take`, {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
  body: "{}",
});

export const listProhibitedCategories = () => request("/api/rules/prohibited");

export const addProhibitedCategory = (csrfToken, category, reason) => request("/api/rules/prohibited", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
  body: JSON.stringify({ category, reason }),
});

export const disableProhibitedCategory = (csrfToken, id) => request(`/api/rules/prohibited/${encodeURIComponent(id)}/disable`, {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
  body: "{}",
});

export const getLedgerRules = () => request("/api/rules/ledger");

export const saveLedgerRules = (csrfToken, limits) => request("/api/rules/ledger", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
  body: JSON.stringify(limits),
});

export const issueEC = (csrfToken, requestData) => request("/api/ledger/issue", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
  body: JSON.stringify(requestData),
});

export const getECBalance = () => request("/api/ledger/balance");

export const listProducts = () => request("/api/products");

export const addProduct = (csrfToken, product) => request("/api/products", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken },
  body: JSON.stringify(product),
});
