import { act, render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { ThemeProvider, useTheme } from "@/components/theme-provider";
import { useShortcutStore } from "@/stores/shortcutStore";
import { useTerminalThemeStore } from "@/stores/terminalThemeStore";

function ThemeValue() {
  return <span>{useTheme().theme}</span>;
}

it("applies a theme changed by a sibling window without remounting", () => {
  localStorage.setItem("theme", "light");
  render(
    <ThemeProvider>
      <ThemeValue />
    </ThemeProvider>
  );
  expect(screen.getByText("light")).toBeVisible();
  act(() => window.dispatchEvent(new StorageEvent("storage", { key: "theme", newValue: "dark" })));
  expect(screen.getByText("dark")).toBeVisible();
  expect(document.documentElement).toHaveClass("dark");
});

it("reloads shortcuts changed in the settings window", () => {
  const binding = { code: "KeyG", mod: true, ctrl: false, shift: true, alt: false };
  localStorage.setItem("keyboard_shortcuts", JSON.stringify({ "page.settings": binding }));
  window.dispatchEvent(new StorageEvent("storage", { key: "keyboard_shortcuts" }));
  expect(useShortcutStore.getState().shortcuts["page.settings"]).toEqual(binding);
});

it("applies terminal appearance changed in a sibling window", async () => {
  useTerminalThemeStore.getState().setFontSize(14);
  const key = useTerminalThemeStore.persist.getOptions().name!;
  const persisted = JSON.parse(localStorage.getItem(key)!);
  persisted.state.fontSize = 19;
  localStorage.setItem(key, JSON.stringify(persisted));
  await act(async () => window.dispatchEvent(new StorageEvent("storage", { key })));
  expect(useTerminalThemeStore.getState().fontSize).toBe(19);
});
