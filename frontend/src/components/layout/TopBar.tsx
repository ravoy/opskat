import React from "react";
import { useTranslation } from "react-i18next";
import { openWorkspace } from "@/lib/openWorkspace";
import { Search, AppWindow } from "lucide-react";
import {
  IconLayoutSidebar,
  IconLayoutSidebarFilled,
  IconLayoutSidebarRight,
  IconLayoutSidebarRightFilled,
} from "@tabler/icons-react";
import { Popover, PopoverContent, PopoverTrigger, Tooltip, TooltipContent, TooltipTrigger, cn } from "@opskat/ui";
import { CommandPalette } from "@/components/command/CommandPalette";
import { usePaletteFocusReturn } from "@/components/command/usePaletteFocusReturn";
import { useShallow } from "zustand/shallow";
import { selectWaitingApprovalConversationIds, useAIStore } from "@/stores/aiStore";
import { WindowTitleBar } from "./WindowTitleBar";
import { usePlatform } from "@/hooks/usePlatform";
import type * as asset_entity from "@bindings/github.com/octays/opskat/internal/model/entity/asset_entity/models";

interface TopBarProps {
  commandOpen: boolean;
  onCommandOpenChange: (open: boolean) => void;
  onConnectAsset: (asset: asset_entity.Asset) => void;
  assetTreeCollapsed: boolean;
  onToggleAssetTree: () => void;
  aiPanelCollapsed: boolean;
  onToggleAIPanel: () => void;
}

export function TopBar({
  commandOpen,
  onCommandOpenChange,
  onConnectAsset,
  assetTreeCollapsed,
  onToggleAssetTree,
  aiPanelCollapsed,
  onToggleAIPanel,
}: TopBarProps) {
  const { t } = useTranslation();
  const approvalCount = useAIStore(useShallow(selectWaitingApprovalConversationIds)).length;

  const platform = usePlatform();
  const onPaletteCloseAutoFocus = usePaletteFocusReturn(commandOpen);

  return (
    <WindowTitleBar>
      {/* 左侧占位，让中间搜索框居中 */}
      <div className="flex-1" />

      {/* 中间：搜索/命令框（按钮形态，点击打开 Popover） */}
      <div
        className="flex w-full max-w-md justify-center"
        style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
      >
        <Popover open={commandOpen} onOpenChange={onCommandOpenChange}>
          <PopoverTrigger asChild>
            <button
              type="button"
              aria-label={t("topBar.searchPlaceholder")}
              className={cn(
                "flex h-7 w-full items-center gap-2 rounded-md border border-input/50 bg-background/60 px-2.5 text-xs",
                "text-muted-foreground transition-colors hover:bg-background hover:text-foreground",
                "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring/45"
              )}
            >
              <Search className="h-4 w-4 shrink-0" />
              <span className="flex-1 truncate text-left">{t("topBar.searchPlaceholder")}</span>
              <kbd className="hidden shrink-0 rounded bg-muted px-1 py-0.5 font-mono text-[11px] sm:inline-block">
                {platform === "darwin" ? "⌘P" : "Ctrl+P"}
              </kbd>
            </button>
          </PopoverTrigger>
          <PopoverContent
            align="center"
            sideOffset={4}
            className="w-[min(640px,90vw)] p-0"
            onOpenAutoFocus={(e) => {
              // 让 CommandPalette 内部的 input 自己取焦
              e.preventDefault();
            }}
            onCloseAutoFocus={onPaletteCloseAutoFocus}
          >
            <CommandPalette
              open={commandOpen}
              onClose={() => onCommandOpenChange(false)}
              onConnectAsset={onConnectAsset}
            />
          </PopoverContent>
        </Popover>
      </div>

      {/* 右侧占位 + 控制按钮 */}
      <div className="flex flex-1 items-center justify-end gap-0.5">
        <div style={{ "--wails-draggable": "no-drag" } as React.CSSProperties} className="flex items-center gap-0.5">
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                data-testid="new-window-button"
                aria-label={t("tab.newWindow")}
                className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent/50 hover:text-foreground"
                onClick={() => void openWorkspace()}
              >
                <AppWindow className="h-4 w-4" />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("tab.newWindow")}</TooltipContent>
          </Tooltip>
          {/* 切换资产列表：实线 = 打开，虚线 = 隐藏 */}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={onToggleAssetTree}
                aria-pressed={!assetTreeCollapsed}
                aria-label={assetTreeCollapsed ? t("topBar.showAssetTree") : t("topBar.hideAssetTree")}
                className={cn(
                  "flex h-7 w-7 items-center justify-center rounded-md transition-colors hover:bg-accent/50",
                  assetTreeCollapsed ? "text-muted-foreground hover:text-foreground" : "text-foreground"
                )}
              >
                {assetTreeCollapsed ? (
                  <IconLayoutSidebar className="h-4 w-4" stroke={1.75} />
                ) : (
                  <IconLayoutSidebarFilled className="h-4 w-4" />
                )}
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom">
              {assetTreeCollapsed ? t("topBar.showAssetTree") : t("topBar.hideAssetTree")}
            </TooltipContent>
          </Tooltip>

          {/* 切换 AI 对话：实线 = 打开，虚线 = 隐藏 */}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={onToggleAIPanel}
                aria-pressed={!aiPanelCollapsed}
                aria-label={aiPanelCollapsed ? t("topBar.showAIPanel") : t("topBar.hideAIPanel")}
                className={cn(
                  "relative flex h-7 w-7 items-center justify-center rounded-md transition-colors hover:bg-accent/50",
                  aiPanelCollapsed ? "text-muted-foreground hover:text-foreground" : "text-foreground"
                )}
              >
                {aiPanelCollapsed ? (
                  <IconLayoutSidebarRight className="h-4 w-4" stroke={1.75} />
                ) : (
                  <IconLayoutSidebarRightFilled className="h-4 w-4" />
                )}
                {approvalCount > 0 && (
                  <span
                    data-testid="ai-panel-approval-badge"
                    role="status"
                    aria-label={t("topBar.pendingApprovals", { count: approvalCount })}
                    className="absolute -right-1 -top-1 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-warning px-1 text-[11px] font-medium leading-none text-warning-foreground"
                  >
                    {approvalCount}
                  </span>
                )}
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom">
              {aiPanelCollapsed ? t("topBar.showAIPanel") : t("topBar.hideAIPanel")}
            </TooltipContent>
          </Tooltip>
        </div>
      </div>
    </WindowTitleBar>
  );
}
