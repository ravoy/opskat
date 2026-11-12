import { getBuiltinPageMeta } from "@/components/layout/pageTabMeta";
import { useTabStore, type Tab } from "@/stores/tabStore";
import { openPageWindow } from "./openPageWindow";
import { openWorkspace } from "./openWorkspace";

type WindowTarget = { assetId: number } | { page: string };

function windowTarget(tab: Tab | undefined): WindowTarget | null {
  if (!tab) return null;
  if ("assetId" in tab.meta && tab.meta.assetId && tab.meta.assetId > 0) return { assetId: tab.meta.assetId };
  if (getBuiltinPageMeta(tab) && tab.meta.type === "page") return { page: tab.meta.pageId };
  return null;
}

export function canOpenTabWindow(tabId: string): boolean {
  return windowTarget(useTabStore.getState().tabs.find((tab) => tab.id === tabId)) !== null;
}

// Opening a window preserves the source tab and its live state.
export async function openTabWindow(tabId: string): Promise<void> {
  const target = windowTarget(useTabStore.getState().tabs.find((tab) => tab.id === tabId));
  if (!target) return;
  if ("assetId" in target) await openWorkspace(target.assetId);
  else await openPageWindow(target);
}
