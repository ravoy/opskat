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
