import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { StopAIGeneration } from "@clients/ai";
import { CloseWindow, EventsEmit, EventsOn, WindowSetTitle } from "@/lib/desktop";
import { isMainWorkspace } from "@/lib/workspace";
import { errorMessage } from "@/lib/errorMessage";
import { openAssetConnection } from "@/lib/openAsset";
import { useAssetStore } from "@/stores/assetStore";
import { openPageTab, openSettingsTab, useTabStore } from "@/stores/tabStore";
import { flushAllConversationsAsync, useAIStore } from "@/stores/aiStore";
import { useSettingsUiStore, type SettingsTab } from "@/stores/settingsUiStore";
import type { PageWindowRequest } from "@bindings/github.com/octays/opskat/internal/desktop/models";
import { resolveTabLabel } from "@/components/layout/pageTabMeta";
import type { QuitActivity } from "@/components/ActiveTasksQuitDialog";

export function useWorkspaceWindow() {
  const { t } = useTranslation();
  const opened = useRef(false);
  const [activities, setActivities] = useState<QuitActivity[] | null>(null);
  const close = useRef<() => Promise<void>>(async () => {});

  useEffect(() => {
    if (isMainWorkspace) return;
    const params = new URLSearchParams(window.location.search);
    const page = params.get("page");
    const focused = !!page || params.get("view") === "connection";
    const navigate = (request: PageWindowRequest) => {
      if (request.page === "settings") {
        if (request.extensionName) useSettingsUiStore.getState().openExtensionDetail(request.extensionName);
        openSettingsTab(t("nav.settings"), (request.section || undefined) as SettingsTab | undefined);
      } else if (request.page !== "ai") {
        openPageTab(request.page);
      }
    };
    const openInitialAsset = () => {
      if (page) {
        if (opened.current) return;
        opened.current = true;
        if (page === "ai") {
          const conversation = Number(params.get("conversation"));
          const ai = useAIStore.getState();
          if (conversation) {
            void ai
              .fetchConversations()
              .then(() => ai.openConversationTab(conversation))
              .catch((error) => toast.error(errorMessage(error)));
          } else {
            ai.openNewConversationTab();
          }
        } else {
          navigate({
            page,
            section: params.get("section") || "",
            extensionName: params.get("extension") || "",
            conversationID: 0,
          });
        }
        return;
      }
      const state = useAssetStore.getState();
      if (!state.initialized || opened.current) return;
      opened.current = true;
      const id = Number(new URLSearchParams(window.location.search).get("asset"));
      if (!id) return;
      const asset = state.assets.find((item) => item.ID === id);
      if (!asset) {
        toast.error(t("tab.windowAssetUnavailable", { id }));
        return;
      }
      void openAssetConnection(asset);
    };
    const updateTitle = () => {
      const state = useTabStore.getState();
      const tab = state.tabs.find((item) => item.id === state.activeTabId);
      if (tab) void WindowSetTitle(`${resolveTabLabel(tab, t)} — OpsKat`);
    };
    openInitialAsset();
    updateTitle();
    const unsubscribeAssets = useAssetStore.subscribe(openInitialAsset);
    let closing = false;
    let preparing = false;
    const acknowledge = () => {
      if (closing && useTabStore.getState().tabs.length === 0) {
        closing = false;
        void EventsEmit("desktop:close-ready");
      }
    };
    const unsubscribeTabs = useTabStore.subscribe((state, previous) => {
      if (state.activeTabId !== previous.activeTabId || state.tabs !== previous.tabs) updateTitle();
      const wasClosing = closing;
      if (state.closeCancellation !== previous.closeCancellation) closing = false;
      acknowledge();
      if (focused && !wasClosing && !preparing && previous.tabs.length > 0 && state.tabs.length === 0)
        void CloseWindow();
    });
    close.current = async () => {
      if (preparing) return;
      preparing = true;
      try {
        const ai = useAIStore.getState();
        await Promise.all(
          Object.entries(ai.conversationStreaming)
            .filter(([, state]) => state.sending)
            .map(([id]) => StopAIGeneration(Number(id)))
        );
        await flushAllConversationsAsync();
        setActivities(null);
        closing = true;
        useTabStore.getState().closeAllTabs();
        acknowledge();
      } catch (error) {
        toast.error(errorMessage(error));
      } finally {
        preparing = false;
      }
    };
    const cancelNavigation = EventsOn("desktop:page-requested", navigate);
    const cancelClose = EventsOn("desktop:close-requested", () => {
      const ai = useAIStore.getState();
      const running = Object.entries(ai.conversationStreaming).filter(([, state]) => state.sending);
      if (running.length) {
        setActivities(
          running.map(([id], index) => ({
            kind: "ai",
            category: "running",
            title:
              ai.conversations.find((conversation) => conversation.ID === Number(id))?.Title ||
              t("appQuit.aiTask", { number: index + 1 }),
          }))
        );
      } else {
        void close.current();
      }
    });
    return () => {
      unsubscribeAssets();
      unsubscribeTabs();
      cancelClose();
      cancelNavigation();
    };
  }, [t]);

  return { activities, cancel: () => setActivities(null), confirm: () => void close.current() };
}
