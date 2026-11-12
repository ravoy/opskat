import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { CancelSSHConnect, ConnectSSHAsync, SplitSSH } from "@clients/ssh";
import { EventsOn } from "@/lib/desktop";
import * as asset_entity from "@bindings/github.com/octays/opskat/internal/model/entity/asset_entity/models";
import { useTabStore, type Tab } from "../tabStore";
import { useAssetStore } from "../assetStore";
import { __resetTerminalSyncListenersForTest, useTerminalStore, type SplitNode } from "../terminalStore";

// 终端窗格的连接状态机:任何一次失败 / 取消之后,窗格里必须有一个可操作的连接条目
// (失败原因 + 重新连接 + 关闭窗格)。分屏树里出现 connecting 节点却没有对应 connections
// 条目,ConnectionProgress 就只能画一个无操作的转圈 —— 这正是本组测试要挡住的状态。
//
// 连接 id 由前端预生成、后端回显(订阅进度事件必须先于连接发起),因此 mock 一律
// 回显请求里的 connectionId,事件与断言都用 store 里 connecting 节点的实际 id。

const handlers = new Map<string, (event: unknown) => void>();

function emit(eventName: string, event: unknown) {
  const handler = handlers.get(eventName);
  if (!handler) throw new Error(`no listener for ${eventName}`);
  handler(event);
}

function terminalTab(id: string, assetId = 1): Tab {
  return {
    id,
    type: "terminal",
    label: `Server ${assetId}`,
    meta: {
      type: "terminal",
      assetId,
      assetName: `Server ${assetId}`,
      assetIcon: "",
      host: "10.0.0.1",
      port: 22,
      username: "root",
    },
  };
}

function sshAsset(id = 1): asset_entity.Asset {
  return new asset_entity.Asset({ ID: id, Name: `Server ${id}`, Type: "ssh", Config: "{}" });
}

function connectingLeaves(node: SplitNode): string[] {
  if (node.type === "connecting") return [node.connectionId];
  if (node.type === "split") return [...connectingLeaves(node.first), ...connectingLeaves(node.second)];
  return [];
}

/** 每个 connecting 节点都必须有连接条目,否则界面只剩无操作的转圈。 */
function expectNoOrphanConnectingNodes() {
  const { tabData, connections } = useTerminalStore.getState();
  for (const data of Object.values(tabData)) {
    for (const id of connectingLeaves(data.splitTree)) {
      expect(connections[id], `connecting node ${id} has no connection entry`).toBeDefined();
    }
  }
}

function onlyConnectingId(tabId: string): string {
  const data = useTerminalStore.getState().tabData[tabId];
  expect(data).toBeDefined();
  const ids = connectingLeaves(data.splitTree);
  expect(ids).toHaveLength(1);
  return ids[0];
}

/** 按当前 connecting 节点的实际 id 发进度事件。 */
function emitConnecting(tabId: string, event: unknown) {
  emit(`ssh:connect:${onlyConnectingId(tabId)}`, event);
}

/** 首次连接时 tab id 就是预生成的连接 id。 */
function onlyTabId(): string {
  const tabs = useTabStore.getState().tabs;
  expect(tabs).toHaveLength(1);
  return tabs[0].id;
}

/** 一个已连上的单窗格标签:tab id 是首个会话 id(connect 成功后 replaceTabId 的结果)。 */
function seedConnectedTab(tabId = "s1") {
  useTabStore.setState({ tabs: [terminalTab(tabId)], activeTabId: tabId });
  useTerminalStore.setState({
    tabData: {
      [tabId]: {
        splitTree: { type: "terminal", sessionId: tabId },
        activePaneId: tabId,
        panes: { [tabId]: { sessionId: tabId, transport: "ssh", connected: true, connectedAt: 1 } },
        directoryFollowMode: "off",
      },
    },
  });
}

const flush = () => new Promise((r) => setTimeout(r, 0));

beforeEach(() => {
  handlers.clear();
  vi.clearAllMocks();
  vi.mocked(EventsOn).mockImplementation(((eventName: string, handler: (event: unknown) => void) => {
    handlers.set(eventName, handler);
    return vi.fn();
  }) as never);
  // 新契约:后端原样采纳前端预生成的 connectionId。
  vi.mocked(ConnectSSHAsync).mockImplementation((req) => Promise.resolve(req.connectionId));
  useAssetStore.setState({ assets: [sshAsset(1)], selectedAssetId: null, initialized: true });
  useTabStore.setState({ tabs: [], activeTabId: null });
  useTerminalStore.setState({ tabData: {}, sessionSync: {}, connections: {}, connectingAssetIds: new Set() });
});

afterEach(() => {
  __resetTerminalSyncListenersForTest();
});

describe("重连失败后重新连接", () => {
  it("在原窗格内重试:新连接接管同一个窗格,不会留下没有连接条目的转圈节点", async () => {
    seedConnectedTab("s1");
    useTerminalStore.getState().reconnect("s1");
    await flush();
    emitConnecting("s1", { type: "error", error: "dial tcp: i/o timeout" });

    const failedId = onlyConnectingId("s1");
    useTerminalStore.getState().retryConnect(failedId);
    await flush();

    expectNoOrphanConnectingNodes();
    expect(useTabStore.getState().tabs.map((t) => t.id)).toEqual(["s1"]);
    const retryId = onlyConnectingId("s1");
    expect(useTerminalStore.getState().connections[retryId].status).toBe("connecting");

    emit(`ssh:connect:${retryId}`, { type: "connected", sessionId: "s4" });
    const data = useTerminalStore.getState().tabData.s1;
    expect(data.splitTree).toEqual({ type: "terminal", sessionId: "s4" });
    expect(data.panes.s4?.connected).toBe(true);
  });

  it("首次连接失败后重试也在原标签内进行,连上后标签换成会话 id", async () => {
    await useTerminalStore.getState().connect(sshAsset(1));
    const firstId = onlyTabId();
    emit(`ssh:connect:${firstId}`, { type: "error", error: "auth failed" });

    useTerminalStore.getState().retryConnect(firstId);
    await flush();

    expectNoOrphanConnectingNodes();
    expect(useTabStore.getState().tabs).toHaveLength(1);
    emitConnecting(onlyTabId(), { type: "connected", sessionId: "s9" });

    expect(useTabStore.getState().tabs.map((t) => t.id)).toEqual(["s9"]);
    expect(useTerminalStore.getState().tabData.s9.splitTree).toEqual({ type: "terminal", sessionId: "s9" });
    expect(useTerminalStore.getState().connections).toEqual({});
  });

  it("重试本身被后端拒绝时,窗格停在失败态并显示原因", async () => {
    seedConnectedTab("s1");
    useTerminalStore.getState().reconnect("s1");
    await flush();
    emitConnecting("s1", { type: "error", error: "timeout" });

    const failedId = onlyConnectingId("s1");
    vi.mocked(ConnectSSHAsync).mockRejectedValueOnce(new Error("资产不存在"));
    useTerminalStore.getState().retryConnect(failedId);
    await flush();

    expectNoOrphanConnectingNodes();
    const id = onlyConnectingId("s1");
    expect(useTerminalStore.getState().connections[id]).toMatchObject({ status: "error", error: "资产不存在" });
  });
});

describe("重连发起即被拒绝", () => {
  it("恢复标签时后端拒绝重连,窗格显示失败原因而不是一块空终端", async () => {
    seedConnectedTab("s1");
    vi.mocked(ConnectSSHAsync).mockRejectedValueOnce(new Error("资产不存在"));

    useTerminalStore.getState().reconnect("s1");
    await flush();

    expectNoOrphanConnectingNodes();
    const id = onlyConnectingId("s1");
    expect(useTerminalStore.getState().connections[id]).toMatchObject({ status: "error", error: "资产不存在" });
  });
});

describe("取消连接", () => {
  it("取消重连后窗格可重试、可关闭,不会停在转圈", async () => {
    seedConnectedTab("s1");
    useTerminalStore.getState().reconnect("s1");
    await flush();

    const connId = onlyConnectingId("s1");
    useTerminalStore.getState().cancelConnect(connId);

    expect(CancelSSHConnect).toHaveBeenCalledWith(connId);
    expectNoOrphanConnectingNodes();
    const id = onlyConnectingId("s1");
    expect(useTerminalStore.getState().connections[id].status).toBe("cancelled");
    expect(useTabStore.getState().tabs.map((t) => t.id)).toEqual(["s1"]);
  });

  it("取消首次连接后标签仍在,可在原处重试", async () => {
    await useTerminalStore.getState().connect(sshAsset(1));
    const connId = onlyTabId();

    useTerminalStore.getState().cancelConnect(connId);

    expect(useTabStore.getState().tabs.map((t) => t.id)).toEqual([connId]);
    expect(useTerminalStore.getState().connections[connId].status).toBe("cancelled");
    expect(useTerminalStore.getState().connectingAssetIds.has(1)).toBe(false);
  });

  it("重试发起期间取消,迟到的新连接被取消且不接管窗格", async () => {
    seedConnectedTab("s1");
    useTerminalStore.getState().reconnect("s1");
    await flush();
    emitConnecting("s1", { type: "error", error: "timeout" });

    const failedId = onlyConnectingId("s1");
    vi.mocked(ConnectSSHAsync).mockImplementationOnce(
      (req) => new Promise<string>((r) => setTimeout(() => r(req.connectionId), 50))
    );
    useTerminalStore.getState().retryConnect(failedId);
    const retryId = onlyConnectingId("s1");
    useTerminalStore.getState().cancelConnect(retryId);
    await flush();

    expect(CancelSSHConnect).toHaveBeenCalledWith(retryId);
    expect(onlyConnectingId("s1")).toBe(retryId);
    expect(useTerminalStore.getState().connections[retryId].status).toBe("cancelled");
  });
});

describe("关闭连接中的窗格", () => {
  it("唯一窗格:关闭即关闭标签", async () => {
    seedConnectedTab("s1");
    useTerminalStore.getState().reconnect("s1");
    await flush();
    emitConnecting("s1", { type: "error", error: "timeout" });

    useTerminalStore.getState().closeConnectionPane(onlyConnectingId("s1"));

    expect(useTabStore.getState().tabs).toEqual([]);
    expect(useTerminalStore.getState().tabData).toEqual({});
    expect(useTerminalStore.getState().connections).toEqual({});
  });

  it("分屏中:只移除该窗格,其余窗格保留并获得焦点", async () => {
    useTabStore.setState({ tabs: [terminalTab("s1")], activeTabId: "s1" });
    useTerminalStore.setState({
      tabData: {
        s1: {
          splitTree: {
            type: "split",
            direction: "horizontal",
            ratio: 0.5,
            first: { type: "terminal", sessionId: "s1" },
            second: { type: "terminal", sessionId: "s2" },
          },
          activePaneId: "s2",
          panes: {
            s1: { sessionId: "s1", transport: "ssh", connected: true, connectedAt: 1 },
            s2: { sessionId: "s2", transport: "ssh", connected: false, connectedAt: 1 },
          },
          directoryFollowMode: "off",
        },
      },
    });
    useTerminalStore.getState().reconnect("s1");
    await flush();
    emitConnecting("s1", { type: "error", error: "timeout" });

    useTerminalStore.getState().closeConnectionPane(onlyConnectingId("s1"));

    const data = useTerminalStore.getState().tabData.s1;
    expect(data.splitTree).toEqual({ type: "terminal", sessionId: "s1" });
    expect(data.activePaneId).toBe("s1");
    expect(useTabStore.getState().tabs.map((t) => t.id)).toEqual(["s1"]);
    expect(useTerminalStore.getState().connections).toEqual({});
  });

  it("关闭终端窗格时,旁边失败的连接窗格算作另一个窗格,标签不被关闭", async () => {
    useTabStore.setState({ tabs: [terminalTab("s1")], activeTabId: "s1" });
    useTerminalStore.setState({
      tabData: {
        s1: {
          splitTree: {
            type: "split",
            direction: "horizontal",
            ratio: 0.5,
            first: { type: "terminal", sessionId: "s1" },
            second: { type: "terminal", sessionId: "s2" },
          },
          activePaneId: "s2",
          panes: {
            s1: { sessionId: "s1", transport: "ssh", connected: true, connectedAt: 1 },
            s2: { sessionId: "s2", transport: "ssh", connected: false, connectedAt: 1 },
          },
          directoryFollowMode: "off",
        },
      },
    });
    useTerminalStore.getState().reconnect("s1");
    await flush();
    emitConnecting("s1", { type: "error", error: "timeout" });
    const failedId = onlyConnectingId("s1");

    useTerminalStore.getState().closePane("s1", "s1");

    expect(useTabStore.getState().tabs.map((t) => t.id)).toEqual(["s1"]);
    const data = useTerminalStore.getState().tabData.s1;
    expect(data.splitTree).toEqual({ type: "connecting", connectionId: failedId });
    expect(data.activePaneId).toBe(failedId);
    expectNoOrphanConnectingNodes();
  });

  it("关闭仍在连接中的窗格会取消后端连接", async () => {
    await useTerminalStore.getState().connect(sshAsset(1));
    const connId = onlyTabId();

    useTerminalStore.getState().closeConnectionPane(connId);

    expect(CancelSSHConnect).toHaveBeenCalledWith(connId);
    expect(useTabStore.getState().tabs).toEqual([]);
  });
});

describe("分屏里按会话重连", () => {
  it("掉线的是非活动窗格时,重连的是掉线的那个窗格", async () => {
    useTabStore.setState({ tabs: [terminalTab("s1")], activeTabId: "s1" });
    useTerminalStore.setState({
      tabData: {
        s1: {
          splitTree: {
            type: "split",
            direction: "horizontal",
            ratio: 0.5,
            first: { type: "terminal", sessionId: "s1" },
            second: { type: "terminal", sessionId: "s2" },
          },
          activePaneId: "s1",
          panes: {
            s1: { sessionId: "s1", transport: "ssh", connected: true, connectedAt: 1 },
            s2: { sessionId: "s2", transport: "ssh", connected: false, connectedAt: 1 },
          },
          directoryFollowMode: "off",
        },
      },
    });

    useTerminalStore.getState().reconnectBySession("s2");
    await flush();

    const retryId = onlyConnectingId("s1");
    const tree = useTerminalStore.getState().tabData.s1.splitTree;
    expect(tree).toMatchObject({
      first: { type: "terminal", sessionId: "s1" },
      second: { type: "connecting", connectionId: retryId },
    });

    // 后台窗格连上后不抢走用户正在用的窗格的焦点。
    emit(`ssh:connect:${retryId}`, { type: "connected", sessionId: "s3" });
    expect(useTerminalStore.getState().tabData.s1.activePaneId).toBe("s1");
  });
});

describe("关闭正在重连的标签", () => {
  it("取消标签内所有在途连接并清掉条目", async () => {
    seedConnectedTab("s1");
    useTerminalStore.getState().reconnect("s1");
    await flush();

    const connId = onlyConnectingId("s1");
    useTabStore.getState().closeTab("s1");

    expect(CancelSSHConnect).toHaveBeenCalledWith(connId);
    expect(useTerminalStore.getState().connections).toEqual({});
  });
});

describe("分屏失败", () => {
  it("新窗格停在失败态并显示原因,可在原处重新连接", async () => {
    useTabStore.setState({ tabs: [terminalTab("s1")], activeTabId: "s1" });
    useTerminalStore.setState({
      tabData: {
        s1: {
          splitTree: { type: "terminal", sessionId: "s1" },
          activePaneId: "s1",
          panes: { s1: { sessionId: "s1", transport: "ssh", connected: true, connectedAt: 1 } },
          directoryFollowMode: "off",
        },
      },
    });
    vi.mocked(SplitSSH).mockRejectedValueOnce(new Error("ssh: open channel: administratively prohibited"));

    useTerminalStore.getState().splitPane("s1", "vertical");
    await flush();

    const failedId = onlyConnectingId("s1");
    const conn = useTerminalStore.getState().connections[failedId];
    expect(conn.status).toBe("error");
    expect(conn.error).toContain("administratively prohibited");
    expectNoOrphanConnectingNodes();

    useTerminalStore.getState().retryConnect(failedId);
    await flush();
    emitConnecting("s1", { type: "connected", sessionId: "s2" });

    const data = useTerminalStore.getState().tabData.s1;
    expect(data.splitTree).toMatchObject({
      type: "split",
      first: { type: "terminal", sessionId: "s1" },
      second: { type: "terminal", sessionId: "s2" },
    });
  });
});
