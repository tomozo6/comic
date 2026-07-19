import { currentUser } from "./firebase.js";

export async function fetchAPI(path) {
  const user = currentUser();
  if (!user) {
    throw new Error("ログインが必要です。");
  }

  const response = await fetch(path, {
    headers: { Authorization: `Bearer ${await user.getIdToken()}` },
  });

  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || "データを取得できませんでした。");
  }

  return response.json();
}
