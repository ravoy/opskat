// Secondary windows keep their session state out of the main window's saved
// workspace. Preferences (theme, shortcuts, etc.) remain shared in localStorage.
export const workspaceId = new URLSearchParams(window.location.search).get("workspace");
export const isMainWorkspace = !workspaceId;
export const workspaceStorage = isMainWorkspace ? localStorage : sessionStorage;

export const focusedWindow =
  new URLSearchParams(window.location.search).has("page") ||
  new URLSearchParams(window.location.search).get("view") === "connection";
