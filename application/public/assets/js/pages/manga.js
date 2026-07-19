import { fetchAPI } from "../api.js";
import { bindLogout, requireSignedIn } from "../auth.js";
import { mangaIDFromPath } from "../routes.js";
import { showError } from "../ui.js";

const title = document.querySelector("#title");
const author = document.querySelector("#author");
const list = document.querySelector("#list");
const status = document.querySelector("#status");

bindLogout(document.querySelector("#logout"));
requireSignedIn(loadManga);

async function loadManga() {
  try {
    const mangaID = mangaIDFromPath();
    const manga = await fetchAPI(`/api/manga/${encodeURIComponent(mangaID)}`);
    title.textContent = manga.title;
    author.textContent = manga.author;
    list.replaceChildren(...manga.volumes.map((volume) => createVolumeLink(manga.id, volume)));
  } catch (error) {
    showError(status, error);
  }
}

function createVolumeLink(mangaID, volume) {
  const link = document.createElement("a");
  link.className = "volume";
  link.href = `/manga/${encodeURIComponent(mangaID)}/volumes/${encodeURIComponent(volume.id)}`;

  const details = document.createElement("span");
  details.append(`第${volume.number}巻`, document.createElement("br"));
  const subtitle = document.createElement("small");
  subtitle.textContent = volume.title;
  details.append(subtitle);

  const action = document.createElement("b");
  action.textContent = "読む →";
  link.append(details, action);
  return link;
}
