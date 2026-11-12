import { isMainWorkspace, workspaceId } from "./workspace";
import { Application, Browser, Clipboard, Events, System, Window } from "@wailsio/runtime";

// Domain consumers receive the payload; the native event envelope stays here.
export function EventsOn<T>(name: string, callback: (data: T) => void): () => void {
  return Events.On(name, (event) => {
    if (
      (name === "desktop:close-requested" ||
        name === "desktop:files-dropped" ||
        name === "ai:flush-all" ||
        name === "desktop:page-requested") &&
      event.sender !== (workspaceId || "main")
    )
      return;
    if (!isMainWorkspace && (name === "app:quit-confirm" || name.startsWith("opsctl:"))) return;
    callback(event.data as T);
  });
}

export const EventsOff = Events.Off;
export const EventsEmit = (name: string, data?: unknown) => Events.Emit(name, data);
export const BrowserOpenURL = Browser.OpenURL;
export const ClipboardGetText = Clipboard.Text;
export const ClipboardSetText = Clipboard.SetText;
export const WindowSetTitle = (title: string) => Window.SetTitle(title);
export const WindowMinimise = () => Window.Minimise();
export const WindowToggleMaximise = () => Window.ToggleMaximise();
export const WindowIsMaximised = () => Window.IsMaximised();
export const WindowIsFullscreen = () => Window.IsFullscreen();
export const Quit = () => Application.Quit();
export const CloseWindow = () => Window.Close();

export async function Environment() {
  const info = await System.Environment();
  return { platform: info.OS, arch: info.Arch, buildType: info.Debug ? "dev" : "production" };
}

interface FileDrop {
  paths: string[];
  x: number;
  y: number;
}

let cancelFileDrop: (() => void) | undefined;
export function OnFileDrop(callback: (x: number, y: number, paths: string[]) => void, _useDropTarget = true) {
  cancelFileDrop = EventsOn<FileDrop>("desktop:files-dropped", ({ paths, x, y }) => callback(x, y, paths));
}
export function OnFileDropOff() {
  cancelFileDrop?.();
  cancelFileDrop = undefined;
}
