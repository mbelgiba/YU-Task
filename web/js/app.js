import {
  addProhibitedCategory,
  devLogin,
  disableProhibitedCategory,
  listMyTasks,
  listProhibitedCategories,
  listProducts,
  listTasks,
  loadAppConfig,
  loadSession,
  logout,
  publishTask,
  addProduct,
  takeTask,
  getLedgerRules,
  saveLedgerRules,
  issueEC,
  getECBalance,
} from "./api.js";
import { currentPage, onRouteChange } from "./router.js";
import { icon } from "./components/icons.js";
import { taskEmpty } from "./components/empty-state.js";

const translations = {
  ru: {
    skip: "Перейти к содержимому", brandSub: "Платформа задач университета", home: "Главная", tasks: "Найти задачу", work: "Моя работа", shop: "Магазин", rules: "Правила задач",
    eyebrow: "Практика начинается здесь", title: "Делайте полезное. Растите на практике.", intro: "Задачи университета, в которых можно применить свои навыки, поработать с командой и получить признание за вклад.", noteTitle: "Для студентов Yessenov University", noteBody: "Выбирайте задачи по интересам и возможностям. Условия, сроки и награда будут указаны заранее.", photo: "[ФОТО: корпус, запросить у заказчика]",
    emptyLabel: "Новые задачи", emptyLabelTasks: "Открытые задачи", emptyLabelWork: "Моя активность", emptyLabelShop: "Каталог", emptyTitle: "Пока задач нет", emptyBody: "Когда подразделения университета опубликуют запросы, они появятся здесь. Загляните позже.", sectionTasks: "Что можно найти", sectionCaption: "Задачи появляются по мере публикации сотрудниками университета.", note1: "Небольшая задача", note1Body: "Короткое поручение, которое можно выполнить самостоятельно.", note2: "Работа по интересам", note2Body: "Выбирайте направление, где хотите получить практический опыт.", note3: "Командный проект", note3Body: "Объединяйтесь с другими студентами для более сложных задач.", notice: "EC — внутренняя бонусная единица университета. Она не отображается в тенге.",
    pageTasks: "Задачи университета", pageWork: "Моя работа", pageShop: "Магазин", pageRules: "Запрещённые категории", pageIntroTasks: "Просматривайте открытые запросы университета и выбирайте подходящие по теме и срокам.", pageIntroWork: "Здесь появятся задачи, за которые вы взялись, и история принятых работ.", pageIntroShop: "Каталог товаров университета. Цены указываются только во внутренних бонусных единицах EC.", pageIntroRules: "Деканат настраивает категории, которые нельзя публиковать. Сервер проверяет каждую новую задачу.",
    pageEmptyTasks: "Пока нет открытых задач", pageEmptyWork: "Вы ещё не взяли задачу", pageEmptyShop: "Каталог пока пуст", emptyTasksBody: "Когда сотрудники университета опубликуют запросы, они появятся в этом списке.", emptyWorkBody: "После того как вы выберете задачу, здесь будут её статус и дальнейшие шаги.", emptyShopBody: "Добавьте подтверждённые товары университета — каталог покажет их здесь.", catalogTitle: "Каталог университета", catalogNote: "Покупка появится после настройки процесса заказов.", addProduct: "Добавить товар", productName: "Название товара", productDescription: "Описание", productCategory: "Раздел каталога", courseCategory: "Курсы", clothingCategory: "Одежда", foodCategory: "Еда в университете", uncategorized: "Без раздела", emptyCategory: "В этом разделе пока нет позиций.", productPrice: "Цена, EC", productStock: "Количество", saveProduct: "Добавить в каталог", noProducts: "В каталоге пока нет товаров.", inStock: "Доступно", outOfStock: "Нет в наличии", productAdded: "Товар добавлен.",
    localLogin: "Войти для разработки", roleLabel: "Роль для локального просмотра", loginButton: "Продолжить локально", signedAs: "Локальный режим", logout: "Выйти", noSession: "Включите YU_DEV_LOGIN=1 для локального входа.", noPermission: "У вашей роли нет доступа к этому разделу.",
    publishTitle: "Опубликовать задачу", titleLabel: "Короткий заголовок", descriptionLabel: "Что нужно сделать", categoryLabel: "Категория", difficultyLabel: "Сложность", easy: "Лёгкая", medium: "Средняя", hard: "Сложная", rewardLabel: "Награда, EC", publishButton: "Опубликовать", takeButton: "Взять задачу", loading: "Загружаем…", loadError: "Не удалось загрузить данные. Попробуйте обновить страницу.", taskPublished: "Задача опубликована.", taskTaken: "Задача добавлена в вашу работу.", noTasks: "Пока задач нет.", status: "Статус", open: "Открыта", taken: "В работе", inReview: "На проверке", accepted: "Принята", saving: "Сохраняем…", formError: "Проверьте поля и попробуйте ещё раз.",
    rulesTitle: "Список запрещённых категорий", categoryNew: "Новая категория", reasonLabel: "Почему публикация запрещена", addRule: "Добавить категорию", disableRule: "Отключить", active: "Активна", disabled: "Отключена", ruleAdded: "Категория добавлена.", ruleDisabled: "Категория отключена.", noRules: "Категории не настроены.", ecRulesTitle: "Внутренние бонусы EC", ecRulesNote: "EC — внутренняя бонусная единица; денежного обмена и вывода нет. Перед начислением задайте оба месячных лимита.", emissionLimit: "Месячный лимит подразделения, EC", studentLimit: "Лимит начисления одному студенту за месяц, EC", saveLimits: "Сохранить лимиты", limitsSaved: "Лимиты сохранены.", issueTitle: "Начислить EC студенту", studentID: "ID учётной записи студента", issueAmount: "Количество EC", issueButton: "Начислить бонусы", issueDone: "Бонусы начислены.", myBalance: "Мой баланс", balanceError: "Баланс пока недоступен.",
    footer: "YU Tasks · Yessenov University", theme: "Сменить тему", language: "Сменить язык"
  },
  en: {
    skip: "Skip to content", brandSub: "University task platform", home: "Home", tasks: "Find a task", work: "My work", shop: "Shop", rules: "Task rules",
    eyebrow: "Practice starts here", title: "Do useful work. Grow through practice.", intro: "University tasks where you can put your skills to use, work with a team, and have your contribution recognized.", noteTitle: "For Yessenov University students", noteBody: "Choose tasks that fit your interests and availability. Scope, deadlines, and rewards will be shown up front.", photo: "[PHOTO: campus building, request from client]",
    emptyLabel: "New tasks", emptyLabelTasks: "Open tasks", emptyLabelWork: "My activity", emptyLabelShop: "Catalog", emptyTitle: "No tasks yet", emptyBody: "Tasks will appear here when university departments publish requests. Check back later.", sectionTasks: "What you can find", sectionCaption: "Tasks appear as university staff publish requests.", note1: "A small task", note1Body: "A short assignment you can complete on your own.", note2: "Work that fits", note2Body: "Choose an area where you would like hands-on experience.", note3: "A team project", note3Body: "Join other students to take on more involved tasks.", notice: "EC is an internal university bonus unit. It is not shown in tenge.",
    pageTasks: "University tasks", pageWork: "My work", pageShop: "Shop", pageRules: "Prohibited categories", pageIntroTasks: "Browse open university requests and find ones that fit your interests and schedule.", pageIntroWork: "Tasks you take on and your accepted work history will appear here.", pageIntroShop: "The university product catalog. Prices are shown only in internal EC bonus units.", pageIntroRules: "The dean office configures categories that cannot be published. The server checks every new task.",
    pageEmptyTasks: "No open tasks yet", pageEmptyWork: "You have not taken a task yet", pageEmptyShop: "The catalog is empty", emptyTasksBody: "University requests will appear here when staff publish them.", emptyWorkBody: "After you choose a task, its status and next steps will show here.", emptyShopBody: "Add approved university products and they will appear in this catalog.", catalogTitle: "University catalog", catalogNote: "Purchases will be available after the order process is set up.", addProduct: "Add a product", productName: "Product name", productDescription: "Description", productCategory: "Catalog section", courseCategory: "Courses", clothingCategory: "Clothing", foodCategory: "Campus food", uncategorized: "Uncategorized", emptyCategory: "There are no items in this section yet.", productPrice: "Price, EC", productStock: "Quantity", saveProduct: "Add to catalog", noProducts: "No products in the catalog yet.", inStock: "Available", outOfStock: "Out of stock", productAdded: "Product added.",
    localLogin: "Local development sign in", roleLabel: "Role for local preview", loginButton: "Continue locally", signedAs: "Local mode", logout: "Sign out", noSession: "Set YU_DEV_LOGIN=1 to enable local sign in.", noPermission: "Your role cannot access this section.",
    publishTitle: "Publish a task", titleLabel: "Short title", descriptionLabel: "What needs to be done", categoryLabel: "Category", difficultyLabel: "Difficulty", easy: "Easy", medium: "Medium", hard: "Hard", rewardLabel: "Reward, EC", publishButton: "Publish task", takeButton: "Take task", loading: "Loading…", loadError: "Could not load data. Please refresh the page.", taskPublished: "Task published.", taskTaken: "Task added to your work.", noTasks: "No tasks yet.", status: "Status", open: "Open", taken: "In progress", inReview: "In review", accepted: "Accepted", saving: "Saving…", formError: "Check the fields and try again.",
    rulesTitle: "Prohibited categories", categoryNew: "New category", reasonLabel: "Why publication is prohibited", addRule: "Add category", disableRule: "Disable", active: "Active", disabled: "Disabled", ruleAdded: "Category added.", ruleDisabled: "Category disabled.", noRules: "No categories configured.", ecRulesTitle: "Internal EC bonuses", ecRulesNote: "EC is an internal bonus unit with no cash exchange or withdrawal. Set both monthly limits before issuing bonuses.", emissionLimit: "Department monthly limit, EC", studentLimit: "Monthly limit per student, EC", saveLimits: "Save limits", limitsSaved: "Limits saved.", issueTitle: "Issue EC to a student", studentID: "Student account ID", issueAmount: "EC amount", issueButton: "Issue bonus", issueDone: "Bonus issued.", myBalance: "My balance", balanceError: "Balance is not available yet.",
    footer: "YU Tasks · Yessenov University", theme: "Change theme", language: "Change language"
  }
};

const nav = ["home", "tasks", "work", "shop"];
let language = localStorage.getItem("yu-language") === "en" ? "en" : "ru";
let appConfig = { devLogin: false };
let session = { authenticated: false };
const t = (key) => translations[language][key] ?? key;
const roles = ["STUDENT", "STAFF", "DEAN_OFFICE", "RECTOR", "PLATFORM_OWNER"];
const roleLabel = (role) => ({ STUDENT: language === "ru" ? "Студент" : "Student", STAFF: language === "ru" ? "Сотрудник" : "Staff", DEAN_OFFICE: language === "ru" ? "Деканат" : "Dean office", RECTOR: language === "ru" ? "Ректор" : "Rector", PLATFORM_OWNER: language === "ru" ? "Владелец платформы" : "Platform owner" })[role] ?? role;
const difficultyLabel = (difficulty) => ({ EASY: t("easy"), MEDIUM: t("medium"), HARD: t("hard") })[difficulty] ?? difficulty;
const canPublish = (role) => ["STAFF", "DEAN_OFFICE", "RECTOR", "PLATFORM_OWNER"].includes(role);
const canManageRules = (role) => ["DEAN_OFFICE", "RECTOR", "PLATFORM_OWNER"].includes(role);

function navLinks(page) {
  const items = session.authenticated && canManageRules(session.role) ? [...nav, "rules"] : nav;
  return items.map((item) => `<li><a class="nav-link" href="#${item}" ${page === item ? 'aria-current="page"' : ""}>${icon(item)}<span>${t(item)}</span></a></li>`).join("");
}

function loginControl() {
  if (session.authenticated) return `<span class="role-chip">${t("signedAs")}: ${roleLabel(session.role)}</span><button class="language-button" id="logout" type="button">${t("logout")}</button>`;
  if (!appConfig.devLogin) return "";
  return `<details class="dev-login"><summary>${t("localLogin")}</summary><form id="login-form"><label for="dev-role">${t("roleLabel")}</label><select id="dev-role" name="role">${roles.map((role) => `<option value="${role}">${roleLabel(role)}</option>`).join("")}</select><button class="primary-button" type="submit">${t("loginButton")}</button><p class="form-message" id="login-message" aria-live="polite"></p></form></details>`;
}

function taskForm() {
  return `<section class="publish-panel"><h2>${t("publishTitle")}</h2><form id="publish-form" class="form-grid">
    <label>${t("titleLabel")}<input name="title" required minlength="4" maxlength="160"></label>
    <label>${t("categoryLabel")}<input name="category" required minlength="2" maxlength="100"></label>
    <label class="wide-field">${t("descriptionLabel")}<textarea name="description" required minlength="10" maxlength="5000" rows="4"></textarea></label>
    <label>${t("difficultyLabel")}<select name="difficulty"><option value="EASY">${t("easy")}</option><option value="MEDIUM">${t("medium")}</option><option value="HARD">${t("hard")}</option></select></label>
    <label>${t("rewardLabel")}<input name="rewardEc" type="number" min="1" step="1" required></label>
    <div class="wide-field"><button class="primary-button" type="submit">${t("publishButton")}</button><p class="form-message" id="publish-message" aria-live="polite"></p></div>
  </form></section>`;
}

function productForm() {
  return `<section class="publish-panel"><h2>${t("addProduct")}</h2><form id="product-form" class="form-grid">
    <label>${t("productCategory")}<select name="category"><option value="COURSE">${t("courseCategory")}</option><option value="CLOTHING">${t("clothingCategory")}</option><option value="CAMPUS_FOOD">${t("foodCategory")}</option></select></label>
    <label>${t("productName")}<input name="name" required minlength="2" maxlength="120"></label>
    <label>${t("productPrice")}<input name="priceEc" type="number" min="1" step="1" required></label>
    <label class="wide-field">${t("productDescription")}<textarea name="description" maxlength="1000" rows="3"></textarea></label>
    <label>${t("productStock")}<input name="stock" type="number" min="0" step="1" required></label>
    <div class="wide-field"><button class="primary-button" type="submit">${t("saveProduct")}</button><p class="form-message" id="product-message" aria-live="polite"></p></div>
  </form></section>`;
}

function pageContent(page) {
  if (page === "home") return `<div class="page"><section class="welcome" aria-labelledby="welcome-title"><div><p class="eyebrow">${t("eyebrow")}</p><h1 id="welcome-title">${t("title")}</h1><p class="intro">${t("intro")}</p><div class="welcome-note"><strong>${t("noteTitle")}</strong><p>${t("noteBody")}</p></div></div><div class="photo-placeholder" role="img" aria-label="${t("photo")}"><span>${t("photo")}</span></div></section>
    <section aria-labelledby="open-tasks-title"><div class="section-head"><div><h2 id="open-tasks-title">${t("sectionTasks")}</h2><p>${t("sectionCaption")}</p></div></div>${taskEmpty(t)}</section>
    <section class="section-gap" aria-label="${t("sectionTasks")}"><div class="three-notes"><article class="note"><span class="note-index">01</span><h3>${t("note1")}</h3><p>${t("note1Body")}</p></article><article class="note"><span class="note-index">02</span><h3>${t("note2")}</h3><p>${t("note2Body")}</p></article><article class="note"><span class="note-index">03</span><h3>${t("note3")}</h3><p>${t("note3Body")}</p></article></div></section><p class="notice">${t("notice")}</p><footer class="footer">${t("footer")} · [ВОПРОС: юр. владелец]</footer></div>`;
  const headingKey = page === "tasks" ? "pageTasks" : page === "work" ? "pageWork" : page === "shop" ? "pageShop" : "pageRules";
  const introKey = page === "tasks" ? "pageIntroTasks" : page === "work" ? "pageIntroWork" : page === "shop" ? "pageIntroShop" : "pageIntroRules";
  if (page === "rules" && (!session.authenticated || !canManageRules(session.role))) return `<div class="page"><header class="subpage-header"><h1>${t(headingKey)}</h1><p class="intro">${t("noPermission")}</p></header></div>`;
  if (page === "rules") return `<div class="page"><header class="subpage-header"><p class="eyebrow">${t("rules")}</p><h1>${t(headingKey)}</h1><p class="intro">${t(introKey)}</p></header>
    <section class="publish-panel"><h2>${t("rulesTitle")}</h2><div id="rules-list" class="task-list"><div class="skeleton"><span class="skeleton-line"></span><span class="skeleton-line"></span></div></div>
    <form id="rule-form" class="form-grid"><label>${t("categoryNew")}<input name="category" required minlength="2" maxlength="100"></label><label>${t("reasonLabel")}<input name="reason" required minlength="3" maxlength="300"></label><div class="wide-field"><button class="primary-button" type="submit">${t("addRule")}</button><p class="form-message" id="rule-message" aria-live="polite"></p></div></form></section>
    <section class="publish-panel"><h2>${t("ecRulesTitle")}</h2><p>${t("ecRulesNote")}</p><form id="ec-limits-form" class="form-grid"><label>${t("emissionLimit")}<input name="monthlyEmissionLimitEc" type="number" min="1" step="1" required></label><label>${t("studentLimit")}<input name="studentMonthlyEarningLimitEc" type="number" min="1" step="1" required></label><div class="wide-field"><button class="primary-button" type="submit">${t("saveLimits")}</button><p class="form-message" id="ec-limits-message" aria-live="polite"></p></div></form>
    ${(session.role === "DEAN_OFFICE" || session.role === "RECTOR") ? `<hr><h3>${t("issueTitle")}</h3><form id="ec-issue-form" class="form-grid"><label>${t("studentID")}<input name="studentId" required maxlength="120"></label><label>${t("issueAmount")}<input name="amountEc" type="number" min="1" step="1" required></label><div class="wide-field"><button class="primary-button" type="submit">${t("issueButton")}</button><p class="form-message" id="ec-issue-message" aria-live="polite"></p></div></form>` : ""}</section></div>`;
  if (page === "shop") return `<div class="page"><header class="subpage-header"><p class="eyebrow">${t("shop")}</p><h1>${t("pageShop")}</h1><p class="intro">${t("pageIntroShop")}</p>${session.authenticated ? `<p class="ec-balance" id="ec-balance" aria-live="polite">${t("loading")}</p>` : ""}</header>
    ${session.authenticated && canPublish(session.role) ? productForm() : ""}<section class="catalog-section" aria-labelledby="catalog-title"><div class="section-head"><h2 id="catalog-title">${t("catalogTitle")}</h2></div>
    <div id="catalog-groups" class="catalog-groups"><div class="skeleton"><span class="skeleton-line"></span><span class="skeleton-line"></span></div></div><p class="catalog-note">${t("catalogNote")}</p></section><footer class="footer">${t("footer")} · [ВОПРОС: юр. владелец]</footer></div>`;
  const titleKey = page === "tasks" ? "pageTasks" : page === "work" ? "pageWork" : "pageShop";
  const intro = t(introKey);
  const canCreate = page === "tasks" && session.authenticated && canPublish(session.role);
  const panel = page === "shop" ? `<section class="task-empty"><div><span class="empty-status">${t("emptyLabelShop")}</span><h2>${t("pageEmptyShop")}</h2><p>${t("emptyShopBody")}</p></div>${icon("shop", "empty-mark-icon")}</section>` : `<div id="task-results" class="task-list"><div class="skeleton"><span class="skeleton-line"></span><span class="skeleton-line"></span></div></div>`;
  return `<div class="page"><header class="subpage-header"><p class="eyebrow">${t(page)}</p><h1>${t(titleKey)}</h1><p class="intro">${intro}</p></header>${canCreate ? taskForm() : ""}${panel}<footer class="footer">${t("footer")} · [ВОПРОС: юр. владелец]</footer></div>`;
}

function element(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function renderTaskCard(task, allowTake, onTaken) {
  const card = element("article", "task-card");
  const meta = element("div", "task-meta", `${task.category} · ${difficultyLabel(task.difficulty)} · ${task.rewardEc} EC`);
  const title = element("h2", "task-card-title", task.title);
  const description = element("p", "task-card-description", task.description);
  const footer = element("div", "task-card-footer");
  const status = element("span", "status-tag", `${t("status")}: ${task.status === "OPEN" ? t("open") : task.status === "TAKEN" ? t("taken") : task.status}`);
  footer.append(status);
  if (allowTake) {
    const button = element("button", "secondary-button", t("takeButton"));
    button.type = "button";
    button.addEventListener("click", async () => {
      button.disabled = true;
      try { await takeTask(session.csrfToken, task.id); onTaken(); }
      catch { button.disabled = false; footer.append(element("span", "form-message", t("formError"))); }
    });
    footer.append(button);
  }
  card.append(meta, title, description, footer);
  return card;
}

async function loadTaskView(page) {
  const container = document.querySelector("#task-results");
  if (!container) return;
  if (!session.authenticated) {
    container.replaceChildren(element("div", "task-empty", appConfig.devLogin ? t("localLogin") : t("noSession")));
    return;
  }
  try {
    const result = page === "work" ? await listMyTasks() : await listTasks();
    if (!result.tasks.length) {
      const key = page === "work" ? "Work" : "Tasks";
      container.replaceChildren(element("div", "task-empty", `${t(`pageEmpty${key}`)}. ${t(`empty${key}Body`)}`));
      return;
    }
    const cards = result.tasks.map((task) => renderTaskCard(task, page === "tasks" && session.role === "STUDENT", () => loadTaskView(page)));
    container.replaceChildren(...cards);
  } catch {
    container.replaceChildren(element("div", "error-state", t("loadError")));
  }
}

async function loadRules() {
  const container = document.querySelector("#rules-list");
  if (!container) return;
  try {
    const [{ categories }, limits] = await Promise.all([listProhibitedCategories(), getLedgerRules()]);
    const limitsForm = document.querySelector("#ec-limits-form");
    if (limitsForm) {
      limitsForm.elements.monthlyEmissionLimitEc.value = limits.monthlyEmissionLimitEc || "";
      limitsForm.elements.studentMonthlyEarningLimitEc.value = limits.studentMonthlyEarningLimitEc || "";
    }
    if (!categories.length) { container.replaceChildren(element("p", "", t("noRules"))); return; }
    container.replaceChildren(...categories.map((rule) => {
      const row = element("article", "rule-row");
      const copy = element("div");
      copy.append(element("strong", "", rule.category), element("p", "", rule.reason), element("span", "rule-state", rule.active ? t("active") : t("disabled")));
      row.append(copy);
      if (rule.active) {
        const button = element("button", "secondary-button", t("disableRule"));
        button.type = "button";
        button.addEventListener("click", async () => {
          button.disabled = true;
          try { await disableProhibitedCategory(session.csrfToken, rule.id); await loadRules(); }
          catch { button.disabled = false; }
        });
        row.append(button);
      }
      return row;
    }));
  } catch { container.replaceChildren(element("div", "error-state", t("loadError"))); }
}

async function loadProducts() {
  const container = document.querySelector("#catalog-groups");
  if (!container) return;
  if (!session.authenticated) {
    container.replaceChildren(element("div", "task-empty", appConfig.devLogin ? t("localLogin") : t("noSession")));
    return;
  }
  try {
    const [{ products }, balance] = await Promise.all([listProducts(), getECBalance()]);
    const balanceNode = document.querySelector("#ec-balance");
    if (balanceNode) balanceNode.textContent = `${t("myBalance")}: ${balance.balanceEc} EC`;
    const groups = ["COURSE", "CLOTHING", "CAMPUS_FOOD", ...(products.some((item) => item.category === "UNCATEGORIZED") ? ["UNCATEGORIZED"] : [])];
    container.replaceChildren(...groups.map((category) => {
      const group = element("section", "catalog-group");
      const heading = category === "COURSE" ? "courseCategory" : category === "CLOTHING" ? "clothingCategory" : category === "CAMPUS_FOOD" ? "foodCategory" : "uncategorized";
      group.append(element("h3", "catalog-heading", t(heading)));
      const items = products.filter((product) => product.category === category);
      if (!items.length) group.append(element("p", "catalog-empty", t("emptyCategory")));
      else {
        const grid = element("div", "catalog-grid");
        grid.append(...items.map((product) => {
          const card = element("article", "product-card");
          card.append(element("p", "product-price", `${product.priceEc} EC`));
          card.append(element("h4", "product-title", product.name));
          if (product.description) card.append(element("p", "task-card-description", product.description));
          card.append(element("p", "product-stock", `${product.stock > 0 ? t("inStock") : t("outOfStock")}: ${product.stock}`));
          return card;
        }));
        group.append(grid);
      }
      return group;
    }));
  } catch { container.replaceChildren(element("div", "error-state", t("loadError"))); }
}

async function render() {
  const app = document.querySelector("#app");
  const page = currentPage();
  app.innerHTML = `<div class="shell"><header class="topbar"><a class="wordmark" href="#home"><span class="wordmark-type">YU Tasks<small>${t("brandSub")}</small></span></a>
    <div class="top-actions">${loginControl()}<button class="language-button" id="language" type="button" aria-label="${t("language")}">${language.toUpperCase()}</button><button class="icon-button" id="theme" type="button" aria-label="${t("theme")}">${icon(document.documentElement.dataset.theme === "dark" ? "sun" : "moon", "nav-icon")}</button></div></header>
    <div class="layout"><aside class="sidebar"><p class="nav-label">YU Tasks</p><nav aria-label="${t("brandSub")}"><ul class="nav-list">${navLinks(page)}</ul></nav></aside><main class="main" id="main" tabindex="-1">${pageContent(page)}</main></div>
    <nav class="bottom-nav" aria-label="${t("brandSub")}"><ul class="nav-list">${navLinks(page)}</ul></nav></div>`;
  document.documentElement.lang = language;
  const skip = document.querySelector(".skip-link");
  if (skip) skip.textContent = t("skip");
  document.querySelector("#language").addEventListener("click", () => { language = language === "ru" ? "en" : "ru"; localStorage.setItem("yu-language", language); render(); });
  document.querySelector("#theme").addEventListener("click", () => { document.documentElement.dataset.theme = document.documentElement.dataset.theme === "dark" ? "light" : "dark"; localStorage.setItem("yu-theme", document.documentElement.dataset.theme); render(); });
  document.querySelector("#login-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const button = event.currentTarget.querySelector("button[type=submit]"); button.disabled = true;
    try { session = await devLogin(new FormData(event.currentTarget).get("role")); render(); }
    catch { document.querySelector("#login-message").textContent = t("formError"); button.disabled = false; }
  });
  document.querySelector("#logout")?.addEventListener("click", async () => { try { await logout(session.csrfToken); session = { authenticated: false }; render(); } catch { /* Keep the active session visible if logout failed. */ } });
  document.querySelector("#publish-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget; const button = form.querySelector("button[type=submit]"); const message = document.querySelector("#publish-message");
    button.disabled = true; button.textContent = t("saving");
    const data = new FormData(form);
    try {
      await publishTask(session.csrfToken, { title: data.get("title"), description: data.get("description"), category: data.get("category"), difficulty: data.get("difficulty"), rewardEc: Number(data.get("rewardEc")) });
      form.reset(); message.textContent = t("taskPublished"); await loadTaskView("tasks");
    } catch { message.textContent = t("formError"); }
    button.disabled = false; button.textContent = t("publishButton");
  });
  document.querySelector("#rule-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget; const message = document.querySelector("#rule-message"); const data = new FormData(form);
    try { await addProhibitedCategory(session.csrfToken, data.get("category"), data.get("reason")); form.reset(); message.textContent = t("ruleAdded"); await loadRules(); }
    catch { message.textContent = t("formError"); }
  });
  document.querySelector("#ec-limits-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget; const data = new FormData(form); const message = document.querySelector("#ec-limits-message");
    try {
      await saveLedgerRules(session.csrfToken, { monthlyEmissionLimitEc: Number(data.get("monthlyEmissionLimitEc")), studentMonthlyEarningLimitEc: Number(data.get("studentMonthlyEarningLimitEc")) });
      message.textContent = t("limitsSaved");
    } catch { message.textContent = t("formError"); }
  });
  document.querySelector("#ec-issue-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget; const data = new FormData(form); const message = document.querySelector("#ec-issue-message"); const button = form.querySelector("button[type=submit]");
    button.disabled = true;
    try {
      const key = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`;
      await issueEC(session.csrfToken, { studentId: data.get("studentId"), amountEc: Number(data.get("amountEc")), idempotencyKey: key });
      message.textContent = t("issueDone"); form.reset();
    } catch { message.textContent = t("formError"); }
    button.disabled = false;
  });
  document.querySelector("#product-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget; const button = form.querySelector("button[type=submit]"); const message = document.querySelector("#product-message"); const data = new FormData(form);
    button.disabled = true;
    try {
      await addProduct(session.csrfToken, { category: data.get("category"), name: data.get("name"), description: data.get("description"), priceEc: Number(data.get("priceEc")), stock: Number(data.get("stock")) });
      form.reset(); message.textContent = t("productAdded"); await loadProducts();
    } catch { message.textContent = t("formError"); }
    button.disabled = false;
  });
  if (page === "tasks" || page === "work") await loadTaskView(page);
  if (page === "rules") await loadRules();
  if (page === "shop") await loadProducts();
}

document.documentElement.dataset.theme = localStorage.getItem("yu-theme") === "dark" ? "dark" : "light";
async function start() {
  try {
    [appConfig, session] = await Promise.all([loadAppConfig(), loadSession()]);
    await render();
  } catch {
    document.querySelector("#app").innerHTML = `<main class="main"><div class="page error-state" role="alert">${language === "ru" ? "Не удалось загрузить приложение. Обновите страницу." : "The app could not load. Please refresh the page."}</div></main>`;
  }
}
start();
onRouteChange(render);
