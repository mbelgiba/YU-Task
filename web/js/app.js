import { loadAppConfig } from "./api.js";
import { currentPage, onRouteChange } from "./router.js";
import { icon } from "./components/icons.js";
import { taskEmpty } from "./components/empty-state.js";

const translations = {
  ru: {
    skip: "Перейти к содержимому", brandSub: "Платформа задач университета", home: "Главная", tasks: "Найти задачу", work: "Моя работа", shop: "Магазин",
    eyebrow: "Практика начинается здесь", title: "Делайте полезное. Растите на практике.", intro: "Задачи университета, в которых можно применить свои навыки, поработать с командой и получить признание за вклад.", noteTitle: "Для студентов Yessenov University", noteBody: "Выбирайте задачи по интересам и возможностям. Условия, сроки и награда будут указаны заранее.", photo: "[ФОТО: корпус, запросить у заказчика]",
    emptyLabel: "Новые задачи", emptyLabelTasks: "Открытые задачи", emptyLabelWork: "Моя активность", emptyLabelShop: "Каталог", emptyTitle: "Пока задач нет", emptyBody: "Когда подразделения университета опубликуют запросы, они появятся здесь. Загляните позже или спросите координатора программы.", sectionTasks: "Что можно найти", sectionCaption: "Задачи появляются по мере публикации сотрудниками университета.", note1: "Небольшая задача", note1Body: "Короткое поручение, которое можно выполнить самостоятельно.", note2: "Работа по интересам", note2Body: "Выбирайте направление, где хотите получить практический опыт.", note3: "Командный проект", note3Body: "Объединяйтесь с другими студентами для более сложных задач.", notice: "EC — внутренняя бонусная единица университета. Баланс и условия начисления будут видны в аккаунте после входа.", pageTasks: "Задачи университета", pageWork: "Моя работа", pageShop: "Магазин", pageIntroTasks: "Просматривайте открытые запросы университета и выбирайте подходящие по теме и срокам.", pageIntroWork: "Здесь появятся задачи, за которые вы взялись, и история принятых работ.", pageIntroShop: "Здесь будут товары и предложения университета, доступные за внутренние бонусы EC.", pageEmptyTasks: "Пока нет открытых задач", pageEmptyWork: "Вы ещё не взяли задачу", pageEmptyShop: "Предложения появятся позже", emptyTasksBody: "Когда сотрудники университета опубликуют запросы, они появятся в этом списке.", emptyWorkBody: "После того как вы выберете задачу, здесь будут её статус и дальнейшие шаги.", emptyShopBody: "Магазин пока не наполнен. Доступные товары и условия появятся после согласования с университетом.", theme: "Сменить тему", language: "Сменить язык", footer: "YU Tasks · Yessenov University"
  },
  en: {
    skip: "Skip to content", brandSub: "University task platform", home: "Home", tasks: "Find a task", work: "My work", shop: "Shop",
    eyebrow: "Practice starts here", title: "Do useful work. Grow through practice.", intro: "University tasks where you can put your skills to use, work with a team, and have your contribution recognized.", noteTitle: "For Yessenov University students", noteBody: "Choose tasks that fit your interests and availability. Scope, deadlines, and rewards will be shown up front.", photo: "[PHOTO: campus building, request from client]",
    emptyLabel: "New tasks", emptyLabelTasks: "Open tasks", emptyLabelWork: "My activity", emptyLabelShop: "Catalog", emptyTitle: "No tasks yet", emptyBody: "Tasks will appear here when university departments publish requests. Check back later or ask the program coordinator.", sectionTasks: "What you can find", sectionCaption: "Tasks appear as university staff publish requests.", note1: "A small task", note1Body: "A short assignment you can complete on your own.", note2: "Work that fits", note2Body: "Choose an area where you would like hands-on experience.", note3: "A team project", note3Body: "Join other students to take on more involved tasks.", notice: "EC is an internal university bonus unit. Your balance and reward terms will be available in your account after sign-in.", pageTasks: "University tasks", pageWork: "My work", pageShop: "Shop", pageIntroTasks: "Browse open university requests and find ones that fit your interests and schedule.", pageIntroWork: "Tasks you take on and your accepted work history will appear here.", pageIntroShop: "University items and offers available for internal EC bonuses will appear here.", pageEmptyTasks: "No open tasks yet", pageEmptyWork: "You have not taken a task yet", pageEmptyShop: "Offers will appear later", emptyTasksBody: "University requests will appear here when staff publish them.", emptyWorkBody: "After you choose a task, its status and next steps will show here.", emptyShopBody: "The shop is not set up yet. Available items and terms will be added after the university approves them.", theme: "Change theme", language: "Change language", footer: "YU Tasks · Yessenov University"
  }
};

const nav = ["home", "tasks", "work", "shop"];
let language = localStorage.getItem("yu-language") === "en" ? "en" : "ru";
const t = (key) => translations[language][key] ?? key;

function navLinks(page) {
  return nav.map((item) => `<li><a class="nav-link" href="#${item}" ${page === item ? 'aria-current="page"' : ""}>${icon(item)}<span>${t(item)}</span></a></li>`).join("");
}

function pageContent(page) {
  if (page === "home") return `<div class="page">
    <section class="welcome" aria-labelledby="welcome-title">
      <div><p class="eyebrow">${t("eyebrow")}</p><h1 id="welcome-title">${t("title")}</h1><p class="intro">${t("intro")}</p>
        <div class="welcome-note"><strong>${t("noteTitle")}</strong><p>${t("noteBody")}</p></div></div>
      <div class="photo-placeholder" role="img" aria-label="${t("photo")}"><span>${t("photo")}</span></div>
    </section>
    <section aria-labelledby="open-tasks-title"><div class="section-head"><div><h2 id="open-tasks-title">${t("sectionTasks")}</h2><p>${t("sectionCaption")}</p></div></div>${taskEmpty(t)}</section>
    <section class="section-gap" aria-label="${t("sectionTasks")}"><div class="three-notes">
      <article class="note"><span class="note-index">01</span><h3>${t("note1")}</h3><p>${t("note1Body")}</p></article>
      <article class="note"><span class="note-index">02</span><h3>${t("note2")}</h3><p>${t("note2Body")}</p></article>
      <article class="note"><span class="note-index">03</span><h3>${t("note3")}</h3><p>${t("note3Body")}</p></article>
    </div></section><p class="notice">${t("notice")}</p>
    <footer class="footer">${t("footer")} · [ВОПРОС: юр. владелец]</footer>
  </div>`;
  const heading = page === "tasks" ? t("pageTasks") : page === "work" ? t("pageWork") : t("pageShop");
  const intro = page === "tasks" ? t("pageIntroTasks") : page === "work" ? t("pageIntroWork") : t("pageIntroShop");
  const pageKey = page === "tasks" ? "Tasks" : page === "work" ? "Work" : "Shop";
  return `<div class="page"><header class="subpage-header"><p class="eyebrow">${t(page)}</p><h1>${heading}</h1><p class="intro">${intro}</p></header>
    <section class="task-empty" aria-labelledby="empty-title"><div><span class="empty-status">${t(`emptyLabel${pageKey}`)}</span><h2 id="empty-title">${t(`pageEmpty${pageKey}`)}</h2><p>${t(`empty${pageKey}Body`)}</p></div><div class="empty-mark">${icon(page === "shop" ? "shop" : page === "work" ? "work" : "search", "")}</div></section>
    <footer class="footer">${t("footer")} · [ВОПРОС: юр. владелец]</footer></div>`;
}

async function render() {
  const app = document.querySelector("#app");
  const page = currentPage();
  app.innerHTML = `<div class="shell">
    <header class="topbar"><a class="wordmark" href="#home"><span class="wordmark-type">YU Tasks<small>${t("brandSub")}</small></span></a>
      <div class="top-actions"><button class="language-button" id="language" type="button" aria-label="${t("language")}">${language.toUpperCase()}</button><button class="icon-button" id="theme" type="button" aria-label="${t("theme")}">${icon(document.documentElement.dataset.theme === "dark" ? "sun" : "moon", "nav-icon")}</button></div>
    </header>
    <div class="layout"><aside class="sidebar"><p class="nav-label">YU Tasks</p><nav aria-label="${t("brandSub")}"><ul class="nav-list">${navLinks(page)}</ul></nav></aside>
      <main class="main" id="main" tabindex="-1">${pageContent(page)}</main></div>
    <nav class="bottom-nav" aria-label="${t("brandSub")}"><ul class="nav-list">${navLinks(page)}</ul></nav>
  </div>`;
  document.documentElement.lang = language;
  const skipLink = document.querySelector(".skip-link");
  if (skipLink) skipLink.textContent = t("skip");
  document.querySelector("#language").addEventListener("click", () => {
    language = language === "ru" ? "en" : "ru";
    localStorage.setItem("yu-language", language);
    render();
  });
  document.querySelector("#theme").addEventListener("click", () => {
    const theme = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("yu-theme", theme);
    render();
  });
}

document.documentElement.dataset.theme = localStorage.getItem("yu-theme") === "dark" ? "dark" : "light";
loadAppConfig().then(render).catch(() => {
  document.querySelector("#app").innerHTML = `<main class="main"><div class="page error-state" role="alert">${language === "ru" ? "Не удалось загрузить приложение. Обновите страницу." : "The app could not load. Please refresh the page."}</div></main>`;
});
onRouteChange(render);
