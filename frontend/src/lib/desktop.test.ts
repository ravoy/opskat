import { beforeEach, describe, expect, it, vi } from "vitest";
const native = vi.hoisted(() => ({ on: vi.fn(), off: vi.fn(), emit: vi.fn() }));
vi.unmock("@/lib/desktop");
vi.mock("@wailsio/runtime", () => ({
  Application: {},
  Browser: {},
  Clipboard: {},
  System: {},
  Window: {},
  Events: { On: native.on, Off: native.off, Emit: native.emit },
}));
vi.mock("./workspace", () => ({ isMainWorkspace: false, workspaceId: "workspace-one" }));
import { EventsOn } from "./desktop";

describe("native event boundary", () => {
  beforeEach(() => native.on.mockReset());
  it("unwraps stream payloads and returns the native subscription cleanup", () => {
    const cleanup = vi.fn();
    native.on.mockReturnValue(cleanup);
    const receive = vi.fn();
    const unsubscribe = EventsOn("ssh:data:session-one", receive);
    native.on.mock.calls[0][1]({ name: "ssh:data:session-one", data: "aGVsbG8=" });
    expect(receive).toHaveBeenCalledWith("aGVsbG8=");
    unsubscribe();
    expect(cleanup).toHaveBeenCalledOnce();
  });
  it("routes close, file drops and save requests to their owning workspace", () => {
    for (const name of ["desktop:close-requested", "desktop:files-dropped", "ai:flush-all", "desktop:page-requested"]) {
      const receive = vi.fn();
      EventsOn(name, receive);
      const handler = native.on.mock.calls.at(-1)![1];
      handler({ name, sender: "main", data: "other" });
      expect(receive).not.toHaveBeenCalled();
      handler({ name, sender: "workspace-one", data: "owned" });
      expect(receive).toHaveBeenCalledExactlyOnceWith("owned");
    }
  });
});
