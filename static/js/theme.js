const savedTheme = localStorage.getItem("theme") || "theme-dark";
document.documentElement.classList.add(savedTheme);

window.addEventListener("DOMContentLoaded", () => {
  const themeSelect = document.getElementById("themeSelect");
  if (themeSelect) {
    themeSelect.value = savedTheme;
  }
});

function setTheme(theme) {
  document.documentElement.classList.remove(
    "theme-light",
    "theme-dark",
    "theme-orange",
  );
  document.documentElement.classList.add(theme);
  localStorage.setItem("theme", theme);

  const themeSelect = document.getElementById("themeSelect");
  if (themeSelect) {
    themeSelect.value = theme;
  }
}
