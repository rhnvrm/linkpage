// LinkPage frontend: hit tracking and admin form validation.
// Bundled by esgun from src/js/app.js -> dist/js/app.js.

// Record a click on any element carrying a data-hit attribute (link cards).
document.addEventListener("click", (e) => {
  const el = e.target.closest("[data-hit]");
  if (!el) {
    return;
  }
  fetch("/hits/" + el.dataset.hit).catch(() => {});
});

// Validate the create-link form before submitting.
document.addEventListener("submit", (e) => {
  const form = e.target;
  if (!form.id || form.id !== "link-create") {
    return;
  }
  const url = form.querySelector('input[name="url"]').value.trim();
  const text = form.querySelector('input[name="text"]').value.trim();
  if (!url || !text) {
    e.preventDefault();
    alert((!url ? "URL" : "Title") + " is required");
  }
});
