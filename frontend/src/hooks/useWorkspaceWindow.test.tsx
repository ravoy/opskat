import { StrictMode, type ReactNode } from "react";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { EventsEmit, EventsOn } from "@/lib/desktop";
import { openAssetConnection } from "@/lib/openAsset";
import { useAssetStore } from "@/stores/assetStore";
import { useTabStore } from "@/stores/tabStore";
import { flushAllConversationsAsync, useAIStore } from "@/stores/aiStore";
import { StopAIGeneration } from "@clients/ai";
import { Asset } from "@bindings/github.com/octays/opskat/internal/model/entity/asset_entity/models";
import { useWorkspaceWindow } from "./useWorkspaceWindow";

vi.mock("@/lib/workspace", () => ({ isMainWorkspace: false, workspaceStorage: sessionStorage }));
vi.mock("@/lib/openAsset", () => ({ openAssetConnection: vi.fn().mockResolvedValue(undefined) }));
vi.mock("@/stores/aiStore", async (original) => ({
  ...(await original<typeof import("@/stores/aiStore")>()),
  flushAllConversationsAsync: vi.fn().mockResolvedValue(undefined),
}));

function requestClose() {
  const callback = [...vi.mocked(EventsOn).mock.calls]
    .reverse()
    .find(([name]) => name === "desktop:close-requested")?.[1];
  if (!callback) throw new Error("workspace did not subscribe to native close");
  callback(undefined);
}

describe("native workspace lifecycle", () => {
  beforeEach(() => {
    window.history.replaceState(null, "", "/?workspace=workspace-test&asset=7");
    useAssetStore.setState({ initialized: true, assets: [new Asset({ ID: 7, Type: "local", Name: "Local" })] });
    useTabStore.setState({
      tabs: [],
      activeTabId: null,
      pendingCloseTabId: null,
      unsavedTabIds: [],
      closeConfirm: null,
      closeCancellation: 0,
    });
    useAIStore.setState({ conversations: [], conversationStreaming: {} });
    vi.mocked(EventsOn).mockClear();
    vi.mocked(EventsOn).mockImplementation(() => vi.fn());
    vi.mocked(EventsEmit).mockClear();
    vi.mocked(openAssetConnection).mockClear();
    vi.mocked(flushAllConversationsAsync).mockClear();
    vi.mocked(StopAIGeneration).mockClear();
  });

  it("starts its asset only once under React StrictMode", async () => {
    renderHook(() => useWorkspaceWindow(), {
      wrapper: ({ children }: { children: ReactNode }) => <StrictMode>{children}</StrictMode>,
    });
    await waitFor(() => expect(openAssetConnection).toHaveBeenCalledOnce());
    expect(vi.mocked(openAssetConnection).mock.calls[0][0].ID).toBe(7);
  });

  it("opens settings in its dedicated shell and handles later section navigation", async () => {
    window.history.replaceState(null, "", "/?workspace=page-settings&page=settings&section=appearance");
    renderHook(() => useWorkspaceWindow());
    expect(useTabStore.getState().tabs.map((tab) => tab.id)).toEqual(["settings"]);
    expect(openAssetConnection).not.toHaveBeenCalled();
    const navigate = vi.mocked(EventsOn).mock.calls.find(([name]) => name === "desktop:page-requested")![1];
    await act(async () => navigate({ page: "settings", section: "shortcuts", extensionName: "", conversationID: 0 }));
    expect(useTabStore.getState().tabs).toHaveLength(1);
    const { useSettingsUiStore } = await import("@/stores/settingsUiStore");
    expect(useSettingsUiStore.getState().activeTab).toBe("shortcuts");
  });

  it("opens a fresh AI conversation only once in a dedicated window", () => {
    window.history.replaceState(null, "", "/?workspace=page-ai-0&page=ai&conversation=0");
    renderHook(() => useWorkspaceWindow(), {
      wrapper: ({ children }: { children: ReactNode }) => <StrictMode>{children}</StrictMode>,
    });
    expect(useTabStore.getState().tabs).toHaveLength(1);
    expect(useTabStore.getState().tabs[0].type).toBe("ai");
    expect(openAssetConnection).not.toHaveBeenCalled();
  });

  it("cancels native closing when an unsaved tab is kept", async () => {
    useTabStore.getState().openTab({
      id: "draft",
      type: "info",
      label: "Draft",
      meta: { type: "info", targetType: "asset", targetId: 7, name: "Draft" },
    });
    useTabStore.getState().setTabUnsaved("draft", true);
    renderHook(() => useWorkspaceWindow());
    await act(async () => requestClose());
    expect(useTabStore.getState().pendingCloseTabId).toBe("draft");
    expect(EventsEmit).not.toHaveBeenCalledWith("desktop:close-ready");
    act(() => useTabStore.getState().cancelPendingClose());
    act(() => useTabStore.getState().forceCloseTab("draft"));
    expect(EventsEmit).not.toHaveBeenCalledWith("desktop:close-ready");
  });

  it("asks before interrupting this workspace's AI and saves before closing", async () => {
    useAIStore.setState({ conversationStreaming: { 10: { sending: true, pendingQueue: [] } } });
    const { result } = renderHook(() => useWorkspaceWindow());
    await act(async () => requestClose());
    expect(result.current.activities).toHaveLength(1);
    expect(StopAIGeneration).not.toHaveBeenCalled();
    await act(async () => result.current.confirm());
    expect(StopAIGeneration).toHaveBeenCalledWith(10);
    expect(flushAllConversationsAsync).toHaveBeenCalledOnce();
    expect(EventsEmit).toHaveBeenCalledWith("desktop:close-ready");
  });
});
