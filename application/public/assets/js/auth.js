import { observeAuth, signOutFromGoogle } from "./firebase.js";

export function redirectSignedInUser() {
  observeAuth((user) => {
    if (user) {
      location.replace("/library");
    }
  });
}

export function requireSignedIn(onSignedIn) {
  observeAuth((user) => {
    if (!user) {
      location.replace("/");
      return;
    }
    void onSignedIn(user);
  });
}

export function bindLogout(button) {
  button.addEventListener("click", () => signOutFromGoogle());
}
