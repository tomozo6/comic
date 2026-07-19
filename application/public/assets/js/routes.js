export function mangaIDFromPath() {
  const [, resource, mangaID] = location.pathname.split("/");
  if (resource !== "manga" || !mangaID) {
    throw new Error("作品を特定できませんでした。");
  }
  return mangaID;
}

export function readerRouteFromPath() {
  const [, resource, mangaID, volumes, volumeID] = location.pathname.split("/");
  if (resource !== "manga" || !mangaID || volumes !== "volumes" || !volumeID) {
    throw new Error("巻を特定できませんでした。");
  }
  return { mangaID, volumeID };
}
