export function showError(status, error) {
  status.textContent = error.message;
  status.classList.add("error");
}

export function clearStatus(status) {
  status.textContent = "";
  status.classList.remove("error");
}
