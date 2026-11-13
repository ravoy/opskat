import { useTranslation } from "react-i18next";
import type { Tab } from "@/stores/tabStore";
import { useTerminalStore } from "@/stores/terminalStore";

/**
 * tab 悬停信息卡：顶部 tab 条与侧边 tab 条共用。终端类 tab 显示资产、地址
 * （ssh 的 user@host:port；本地/串口显示通道）与连接状态；查询类显示资产、
 * 类型与驱动。其余类型只有标签本身时不再额外渲染。
 */
export function TabHoverCard({ tab }: { tab: Tab }) {
  const { t } = useTranslation();
  const panes = useTerminalStore((s) => (tab.type === "terminal" ? s.tabData[tab.id]?.panes : undefined));

  if (tab.type === "editor") {
    const m = tab.meta as { assetName: string; remotePath: string };
    return (
      <div className="space-y-1" data-testid="tab-hover-card">
        <div className="flex items-baseline gap-2 text-xs">
          <span className="shrink-0 text-muted-foreground">{t("tab.hover.asset")}</span>
          <span>{m.assetName || tab.label}</span>
        </div>
        <div className="flex items-baseline gap-2 text-xs">
          <span className="shrink-0 text-muted-foreground">{t("tab.hover.remotePath")}</span>
          <span className="font-mono break-all">{m.remotePath}</span>
        </div>
      </div>
    );
  }
  if (tab.type !== "terminal" && tab.type !== "query") return <div className="text-xs">{tab.label}</div>;

  const meta = tab.meta;
  const rows: Array<{ label: string; value: string; mono?: boolean }> = [];

  if (tab.type === "terminal") {
    const m = meta as { assetName: string; host: string; port: number; username: string };
    rows.push({ label: t("tab.hover.asset"), value: m.assetName || tab.label });
    if (m.host) {
      const address = m.username ? `${m.username}@${m.host}:${m.port}` : `${m.host}:${m.port}`;
      rows.push({ label: t("tab.hover.address"), value: address, mono: true });
    } else {
      rows.push({ label: t("tab.hover.address"), value: t("tab.hover.localChannel") });
    }
    const list = Object.values(panes ?? {});
    const state = list.length === 0 ? "connecting" : list.every((p) => p.connected) ? "connected" : "disconnected";
    rows.push({ label: t("tab.hover.status"), value: t(`tab.hover.status_${state}`) });
  } else {
    const m = meta as { assetName: string; assetType: string; driver?: string; defaultDatabase?: string };
    rows.push({ label: t("tab.hover.asset"), value: m.assetName || tab.label });
    rows.push({ label: t("tab.hover.assetType"), value: m.assetType });
    if (m.driver) rows.push({ label: t("tab.hover.driver"), value: m.driver, mono: true });
    if (m.defaultDatabase) rows.push({ label: t("tab.hover.database"), value: m.defaultDatabase, mono: true });
  }

  return (
    <div className="space-y-1" data-testid="tab-hover-card">
      {rows.map((row) => (
        <div key={row.label} className="flex items-baseline gap-2 text-xs">
          <span className="shrink-0 text-muted-foreground">{row.label}</span>
          <span className={row.mono ? "font-mono" : undefined}>{row.value}</span>
        </div>
      ))}
    </div>
  );
}
