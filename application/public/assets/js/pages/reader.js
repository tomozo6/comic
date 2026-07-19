import { fetchAPI } from "../api.js";
import { bindLogout, requireSignedIn } from "../auth.js";
import { readerRouteFromPath } from "../routes.js";
import { showError } from "../ui.js";

const back = document.querySelector("#back");
const title = document.querySelector("#title");
const pages = document.querySelector("#pages");
const status = document.querySelector("#status");

bindLogout(document.querySelector("#logout"));
requireSignedIn(() => loadReader());

async function loadReader(retry = false) {
  try {
    const { mangaID, volumeID } = readerRouteFromPath();
    back.href = `/manga/${encodeURIComponent(mangaID)}`;
    const volume = await fetchAPI(
      `/api/manga/${encodeURIComponent(mangaID)}/volumes/${encodeURIComponent(volumeID)}`,
    );
    title.textContent = `第${volume.volumeNumber}巻 ${volume.volumeTitle}`;
    pages.replaceChildren(...volume.pages.map((page) => createPageImage(volume, page, retry)));
  } catch (error) {
    showError(status, error);
  }
}

function createPageImage(volume, page, retry) {
  const image = new Image();
  image.className = "page";
  image.src = page.imageUrl;
  image.alt = `${volume.volumeTitle} ${page.number}ページ`;
  image.addEventListener("error", () => {
    if (!retry) {
      void loadReader(true);
    }
  });
  return image;
}
