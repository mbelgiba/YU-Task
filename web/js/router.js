const pages = new Set(["home", "tasks", "work", "shop", "rules"]);

export function currentPage() {
  const route = location.hash.replace(/^#\/?/, "");
  return pages.has(route) ? route : "home";
}

export function onRouteChange(callback) {
  window.addEventListener("hashchange", callback);
}
