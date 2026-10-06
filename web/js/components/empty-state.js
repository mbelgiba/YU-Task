import { icon } from "./icons.js";

export function taskEmpty(t) {
  return `<section class="task-empty" aria-labelledby="empty-title">
    <div><span class="empty-status">${t("emptyLabel")}</span><h2 id="empty-title">${t("emptyTitle")}</h2>
      <p>${t("emptyBody")}</p></div>
    <div class="empty-mark">${icon("search", "")}</div>
  </section>`;
}
