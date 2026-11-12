import { expect, type Page } from "@playwright/test";

/**
 * Opens the settings page on one section of its left navigation. `section` is the
 * nav item's id (`settings-nav-<id>` in SettingsPage.tsx: "extensions",
 * "custom-types", …), so the helper works whatever language the spec runs in.
 */
export async function openSettingsSection(page: Page, section: string): Promise<void> {
  if (new URL(page.url()).searchParams.get("page") !== "settings") {
    await page.getByTestId("nav-settings").click();
    // Mirror the real settings WebView in Playwright; IPC retains its native name.
    await expect.poll(async () => (await page.request.get("/wails/verification-windows")).json()).toContain("page-settings");
    await page.goto("/?workspace=page-settings&page=settings");
  }
  const item = page.getByTestId(`settings-nav-${section}`);
  await item.click();
  await expect(item).toHaveAttribute("aria-current", "page");
}
