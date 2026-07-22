import { fetchAPI } from "../api.js";
import { bindLogout, requireSignedIn } from "../auth.js";
import { readerRouteFromPath } from "../routes.js";
import { showError } from "../ui.js";

const back = document.querySelector("#back");
const title = document.querySelector("#title");
const pages = document.querySelector("#pages");
const status = document.querySelector("#status");
const pageURLBatchSize = 8;
const batchPrefetchThreshold = 3;
let pageObserver;
let reader;

bindLogout(document.querySelector("#logout"));
requireSignedIn(loadReader);

async function loadReader() {
  try {
    const { mangaID, volumeID } = readerRouteFromPath();
    back.href = `/manga/${encodeURIComponent(mangaID)}`;
    const volume = await fetchAPI(
      `/api/manga/${encodeURIComponent(mangaID)}/volumes/${encodeURIComponent(volumeID)}`,
    );
    title.textContent = `第${volume.volumeNumber}巻 ${volume.volumeTitle}`;
    reader = {
      mangaID,
      volumeID,
      pageCount: volume.pageCount,
      pageURLs: new Map(volume.pages.map((page) => [page.number, page.imageUrl])),
      batches: new Set([1]),
      pendingBatches: new Map(),
    };
    renderPages(volume);
  } catch (error) {
    showError(status, error);
  }
}

function renderPages(volume) {
  pageObserver?.disconnect();
  const placeholders = Array.from({ length: volume.pageCount }, (_, index) =>
    createPagePlaceholder(volume, index + 1),
  );
  pages.replaceChildren(...placeholders);

  if (!("IntersectionObserver" in window)) {
    placeholders.forEach((placeholder) => void ensurePageImage(placeholder));
    return;
  }

  pageObserver = new IntersectionObserver(
    (entries, observer) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          void ensurePageImage(entry.target);
          observer.unobserve(entry.target);
        }
      });
    },
    { rootMargin: "300% 0px" },
  );
  placeholders.forEach((placeholder) => pageObserver.observe(placeholder));
}

function createPagePlaceholder(volume, pageNumber) {
  const placeholder = document.createElement("div");
  placeholder.className = "page-placeholder";
  placeholder.setAttribute("aria-busy", "true");
  placeholder.dataset.pageNumber = pageNumber;

  const image = new Image();
  image.className = "page";
  image.alt = `${volume.volumeTitle} ${pageNumber}ページ`;
  image.decoding = "async";
  placeholder.append(image);
  return placeholder;
}

async function ensurePageImage(placeholder) {
  const pageNumber = Number(placeholder.dataset.pageNumber);
  const batchStart = pageBatchStart(pageNumber);
  try {
    await fetchPageBatch(batchStart);
  } catch (error) {
    showError(status, error);
    return;
  }
  loadPageImage(placeholder, reader.pageURLs.get(pageNumber));
  prefetchNextBatch(pageNumber);
}

function pageBatchStart(pageNumber) {
  return Math.floor((pageNumber - 1) / pageURLBatchSize) * pageURLBatchSize + 1;
}

function fetchPageBatch(start, refresh = false) {
  if (!refresh && reader.batches.has(start)) {
    return Promise.resolve();
  }
  if (reader.pendingBatches.has(start)) {
    return reader.pendingBatches.get(start);
  }
  const request = fetchAPI(
    `/api/manga/${encodeURIComponent(reader.mangaID)}/volumes/${encodeURIComponent(reader.volumeID)}/pages?start=${start}`,
  )
    .then((batch) => {
      batch.pages.forEach((page) => reader.pageURLs.set(page.number, page.imageUrl));
      reader.batches.add(start);
    })
    .finally(() => reader.pendingBatches.delete(start));
  reader.pendingBatches.set(start, request);
  return request;
}

function prefetchNextBatch(pageNumber) {
  const currentBatchStart = pageBatchStart(pageNumber);
  const prefetchAt = currentBatchStart + pageURLBatchSize - batchPrefetchThreshold;
  const nextBatchStart = currentBatchStart + pageURLBatchSize;
  if (pageNumber >= prefetchAt && nextBatchStart <= reader.pageCount) {
    void fetchPageBatch(nextBatchStart).catch((error) => showError(status, error));
  }
}

function loadPageImage(placeholder, imageURL) {
  if (placeholder.dataset.loaded) {
    return;
  }
  placeholder.dataset.loaded = "true";
  const image = placeholder.querySelector("img");
  image.addEventListener(
    "load",
    () => {
      placeholder.classList.add("is-loaded");
      placeholder.removeAttribute("aria-busy");
    },
    { once: true },
  );
  image.addEventListener("error", () => {
    void refreshPageImage(placeholder);
  }, { once: true });
  image.src = imageURL;
}

async function refreshPageImage(placeholder) {
  if (placeholder.dataset.retried) {
    return;
  }
  placeholder.dataset.retried = "true";
  const pageNumber = Number(placeholder.dataset.pageNumber);
  try {
    await fetchPageBatch(pageBatchStart(pageNumber), true);
    placeholder.querySelector("img").src = reader.pageURLs.get(pageNumber);
  } catch (error) {
    showError(status, error);
  }
}
