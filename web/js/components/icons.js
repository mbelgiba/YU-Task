export function icon(name, className = "nav-icon") {
  const safeName = ["home", "tasks", "work", "shop", "search", "moon", "sun", "rules"].includes(name) ? name : "home";
  return `<svg class="${className}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><use href="/assets/icons/sprite.svg#${safeName}"></use></svg>`;
}
