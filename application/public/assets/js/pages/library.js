import { fetchAPI } from "../api.js";
import { bindLogout, requireSignedIn } from "../auth.js";
import { showError } from "../ui.js";

const list = document.querySelector("#list");
const status = document.querySelector("#status");

bindLogout(document.querySelector("#logout"));
requireSignedIn(loadLibrary);

async function loadLibrary() {
  try {
    const mangas = await fetchAPI("/api/manga");
    list.replaceChildren(...mangas.map(createMangaLink));
  } catch (error) {
    showError(status, error);
  }
}

function createMangaLink(manga) {
  const link = document.createElement("a");
  link.className = "manga";
  link.href = `/manga/${encodeURIComponent(manga.id)}`;

  const cover = document.createElement("span");
  cover.className = "cover";
  cover.textContent = "本";

  const details = document.createElement("span");
  const title = document.createElement("h2");
  title.textContent = manga.title;
  const author = document.createElement("p");
  author.textContent = manga.author;
  details.append(title, author);

  link.append(cover, details);
  return link;
}
