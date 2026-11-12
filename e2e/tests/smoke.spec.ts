import { test, expect } from "@playwright/test";
import { nativeTool } from "../fixtures/native";

const TOOLS = ["forward", "sshkeys", "snippets", "audit"] as const;

test("management stays in tabs and opens native windows only on request", async ({ page, request }) => {
  await page.goto("/");
  await expect(page.getByTestId("app-root")).toBeVisible();
  for (const id of TOOLS) {
    const button = page.getByTestId(`nav-${id}`);
    const before = (await (await request.get("/wails/verification-windows")).json()).sort();
    await button.click();
    await expect(button).toHaveAttribute("data-active", "true");
    expect((await (await request.get("/wails/verification-windows")).json()).sort()).toEqual(before);
    const label = (await button.getAttribute("aria-label"))!;
    await page.getByTitle(label, { exact: true }).first().click({ button: "right" });
    await page.getByRole("menuitem", { name: /Open in New Window|在新窗口打开/ }).click();
    const name = `page-${id}`;
    await expect.poll(async () => (await request.get("/wails/verification-windows")).json()).toContain(name);
    try {
      await expect.poll(async () => nativeTool(request, "js_eval", {
        window: name,
        js: "return {mounted:!!document.querySelector('[data-testid=app-root]'),navigation:!!document.querySelector('[data-testid=nav-home]'),loading:!!document.querySelector('.animate-spin')};",
      })).toEqual({ mounted: true, navigation: false, loading: false });
      await page.getByTitle(label, { exact: true }).first().click({ button: "right" });
      await page.getByRole("menuitem", { name: /Open in New Window|在新窗口打开/ }).click();
      expect((await (await request.get("/wails/verification-windows")).json()).filter((item: string) => item === name)).toHaveLength(1);
    } finally {
      await nativeTool(request, "js_eval", {
        window: name,
        js: "const {CloseWindow}=await import('/src/lib/desktop.ts');setTimeout(()=>CloseWindow(),100);return true;",
      });
      await expect.poll(async () => (await request.get("/wails/verification-windows")).json()).not.toContain(name);
    }
    await expect(button).toHaveAttribute("data-active", "true");
  }
});
