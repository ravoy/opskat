import { render, screen } from "@testing-library/react";
import { describe, it, expect, beforeEach } from "vitest";
import { TabHoverCard } from "../TabHoverCard";
import { useTerminalStore } from "@/stores/terminalStore";
import type { Tab } from "@/stores/tabStore";

// tab 悬停卡的契约：ssh 终端显示 user@host:port 与连接状态；本地/串口没有 host 时
// 显示通道说明；查询类显示资产/类型/驱动；编辑器显示资产与远程路径。

function terminalTab(overrides: Record<string, unknown> = {}): Tab {
  return {
    id: "t1",
    type: "terminal",
    label: "web-01",
    meta: {
      type: "terminal",
      assetId: 1,
      assetName: "web-01",
      assetIcon: "",
      host: "10.0.0.1",
      port: 22,
      username: "root",
    },
    ...overrides,
  } as Tab;
}

describe("TabHoverCard", () => {
  beforeEach(() => {
    useTerminalStore.setState({ tabData: {}, sessionSync: {}, connections: {}, connectingAssetIds: new Set() });
  });

  it("ssh 终端：资产 + user@host:port + 连接状态", () => {
    useTerminalStore.setState({
      tabData: {
        t1: {
          splitTree: { type: "terminal", sessionId: "s1" },
          activePaneId: "s1",
          panes: { s1: { sessionId: "s1", transport: "ssh", connected: true, connectedAt: 1 } },
          directoryFollowMode: "off",
        },
      },
    });
    render(<TabHoverCard tab={terminalTab()} />);
    expect(screen.getByText("tab.hover.asset")).toBeInTheDocument();
    expect(screen.getByText("root@10.0.0.1:22")).toBeInTheDocument();
    expect(screen.getByText("tab.hover.status_connected")).toBeInTheDocument();
  });

  it("断开的终端显示断开状态", () => {
    useTerminalStore.setState({
      tabData: {
        t1: {
          splitTree: { type: "terminal", sessionId: "s1" },
          activePaneId: "s1",
          panes: { s1: { sessionId: "s1", transport: "ssh", connected: false, connectedAt: 1 } },
          directoryFollowMode: "off",
        },
      },
    });
    render(<TabHoverCard tab={terminalTab()} />);
    expect(screen.getByText("tab.hover.status_disconnected")).toBeInTheDocument();
  });

  it("本地/串口终端没有 host：显示通道说明而不是空地址", () => {
    render(
      <TabHoverCard
        tab={terminalTab({
          meta: { type: "terminal", assetId: 2, assetName: "local", assetIcon: "", host: "", port: 0, username: "" },
        })}
      />
    );
    expect(screen.getByText("tab.hover.localChannel")).toBeInTheDocument();
    expect(screen.getByText("tab.hover.status_connecting")).toBeInTheDocument();
  });

  it("查询 tab：资产 + 类型 + 驱动 + 默认库", () => {
    render(
      <TabHoverCard
        tab={
          {
            id: "q1",
            type: "query",
            label: "pg-prod",
            meta: {
              type: "query",
              assetId: 3,
              assetName: "pg-prod",
              assetIcon: "",
              assetType: "database",
              driver: "postgresql",
              defaultDatabase: "app",
            },
          } as Tab
        }
      />
    );
    expect(screen.getByText("pg-prod")).toBeInTheDocument();
    expect(screen.getByText("tab.hover.assetType")).toBeInTheDocument();
    expect(screen.getByText("postgresql")).toBeInTheDocument();
    expect(screen.getByText("app")).toBeInTheDocument();
  });

  it("编辑器 tab：资产 + 远程路径", () => {
    render(
      <TabHoverCard
        tab={
          {
            id: "e1",
            type: "editor",
            label: "app.conf",
            meta: { assetName: "web-01", remotePath: "/etc/app/app.conf" },
          } as Tab
        }
      />
    );
    expect(screen.getByText("web-01")).toBeInTheDocument();
    expect(screen.getByText("/etc/app/app.conf")).toBeInTheDocument();
  });
});
