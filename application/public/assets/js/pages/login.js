import { redirectSignedInUser } from "../auth.js";
import { signInWithGoogle } from "../firebase.js";
import { showError } from "../ui.js";

const loginButton = document.querySelector("#login");
const status = document.querySelector("#status");

redirectSignedInUser();

loginButton.addEventListener("click", async () => {
  try {
    await signInWithGoogle();
  } catch {
    showError(status, new Error("ログインできませんでした。"));
  }
});
