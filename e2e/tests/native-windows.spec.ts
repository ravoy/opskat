import { test, expect } from "@playwright/test";

import { nativeTool } from "../fixtures/native";

test("native workspaces render and close independently", async ({ page, request }) => {
  await page.goto("/");
  await expect(page.getByTestId("app-root")).toBeVisible();
  const main = await nativeTool(request, "js_eval", {
    window: "main",
    js: "const root = document.querySelector('[data-testid=app-root]'); return {rendered: !!root && root.getBoundingClientRect().height > 0};",
  });
  expect(main.rendered).toBe(true);
  const previous: string[] = await (await request.get("/wails/verification-windows")).json();
  await page.getByTestId("new-window-button").click();
  let child = "";
  await expect.poll(async () => {
    const windows: string[] = await (await request.get("/wails/verification-windows")).json();
    child = windows.find((name) => name.startsWith("workspace-") && !previous.includes(name)) ?? "";
    return !!child;
  }).toBe(true);
  try {
    await expect.poll(async () => nativeTool(request, "js_eval", {
      window: child,
      js: "return {rendered:!!document.querySelector('[data-testid=app-root]'),workspace:new URLSearchParams(location.search).get('workspace')};",
    })).toEqual({ rendered: true, workspace: child });
  } finally {
    await nativeTool(request, "js_eval", {
      window: child,
      js: "const {CloseWindow} = await import('/src/lib/desktop.ts'); setTimeout(() => CloseWindow(), 100); return true;",
    });
    await expect.poll(async () => (await request.get("/wails/verification-windows")).json()).not.toContain(child);
  }
  await expect(page.getByTestId("app-root")).toBeVisible();
});

// A tool page window (settings) is a form/list surface: it opens smaller than an
// asset workspace and never below its minimum (windowSizeFor in windows.go).
test("settings window opens at the tool-page size", async ({ page, request }) => {
  await page.goto("/");
  await expect(page.getByTestId("app-root")).toBeVisible();

  await page.getByTestId("nav-settings").click();
  await expect.poll(async () => (await request.get("/wails/verification-windows")).json()).toContain("page-settings");

  const size = await nativeTool(request, "js_eval", {
    window: "page-settings",
    js: "return {w: innerWidth, h: innerHeight, page: new URLSearchParams(location.search).get('page')};",
  });
  expect(size.page).toBe("settings");
  // 1080×720 native, minus the macOS title bar inset the webview reports as innerHeight.
  expect(size.w).toBe(1080);
  expect(size.h).toBeGreaterThan(640);
  expect(size.h).toBeLessThanOrEqual(720);

  await nativeTool(request, "js_eval", {
    window: "page-settings",
    js: "const {CloseWindow} = await import('/src/lib/desktop.ts'); setTimeout(() => CloseWindow(), 100); return true;",
  });
  await expect.poll(async () => (await request.get("/wails/verification-windows")).json()).not.toContain("page-settings");
});
