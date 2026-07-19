import { initializeApp } from "https://www.gstatic.com/firebasejs/10.12.5/firebase-app.js";
import {
  getAuth,
  GoogleAuthProvider,
  onAuthStateChanged,
  signInWithPopup,
  signOut,
} from "https://www.gstatic.com/firebasejs/10.12.5/firebase-auth.js";

// const firebaseConfig = {
//   apiKey: "AIzaSyCk4yaphRmQyDILB7ab_aCqS0q2fZv1y8A",
//   authDomain: "manga-stg.firebaseapp.com",
//   projectId: "manga-stg",
//   appId: "1:695648038080:web:14d5b47a6262d560e4d125",
// };

const firebaseConfig = {
  apiKey: "AIzaSyAfy-qizJ10IVaDgKnzVEDUtgo7_nYnGLE",
  authDomain: "tomozo6.firebaseapp.com",
  projectId: "tomozo6",
  appId: "1:673020427466:web:c00b4ca05a124202430833",
};




const auth = getAuth(initializeApp(firebaseConfig));
const googleProvider = new GoogleAuthProvider();
googleProvider.setCustomParameters({ prompt: "select_account" });

export function observeAuth(callback) {
  return onAuthStateChanged(auth, callback);
}

export function signInWithGoogle() {
  return signInWithPopup(auth, googleProvider);
}

export function signOutFromGoogle() {
  return signOut(auth);
}

export function currentUser() {
  return auth.currentUser;
}
