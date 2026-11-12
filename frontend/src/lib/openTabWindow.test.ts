import { beforeEach, expect, it, vi } from "vitest";
import { OpenPageWindow, OpenWorkspaceWindow } from "@clients/system";
import { useTabStore, type Tab } from "@/stores/tabStore";
import { canOpenTabWindow, openTabWindow } from "./openTabWindow";

beforeEach(() => {
  vi.mocked(OpenPageWindow).mockClear();
  vi.mocked(OpenWorkspaceWindow).mockClear();
  useTabStore.setState({ tabs: [], activeTabId: null });
});

it("opens an explicitly selected management tab in a window and retains its source", async () => {
  const tab: Tab = { id: "audit", type: "page", label: "Audit", meta: { type: "page", pageId: "audit" } };
  useTabStore.getState().openTab(tab);
  expect(canOpenTabWindow(tab.id)).toBe(true);
  await openTabWindow(tab.id);
  expect(OpenPageWindow).toHaveBeenCalledWith(expect.objectContaining({ page: "audit" }));
  expect(useTabStore.getState().tabs).toEqual([tab]);
});

it("starts an independent asset connection without removing the current session", async () => {
  const tab: Tab = {
    id: "terminal",
    type: "page",
    label: "Desktop",
    meta: { type: "page", pageId: "vnc", assetId: 8 },
  };
  useTabStore.getState().openTab(tab);
  await openTabWindow(tab.id);
  expect(OpenWorkspaceWindow).toHaveBeenCalledWith(8);
  expect(useTabStore.getState().tabs).toEqual([tab]);
});

it("does not offer a window for an unsaved AI draft or a closed tab", async () => {
  useTabStore
    .getState()
    .openTab({ id: "draft", type: "ai", label: "Draft", meta: { type: "ai", conversationId: null, title: "Draft" } });
  expect(canOpenTabWindow("draft")).toBe(false);
  await openTabWindow("missing");
  expect(OpenPageWindow).not.toHaveBeenCalled();
  expect(OpenWorkspaceWindow).not.toHaveBeenCalled();
});
