import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useWailsEvent } from "../hooks/useWailsEvent";
import { EventsOn } from "@/lib/desktop";

// 订阅与退订都走监听者自己的 cancel：EventsOff(name) 会把这个事件名下的所有监听
// 一起摘掉（同事件名的另一个组件会突然失联）。

describe("useWailsEvent", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("subscribes to event on mount", () => {
    const handler = vi.fn();
    renderHook(() => useWailsEvent("test:event", handler));

    expect(EventsOn).toHaveBeenCalledWith("test:event", handler);
  });

  it("unsubscribes with this listener's own cancel", () => {
    const cancel = vi.fn();
    vi.mocked(EventsOn).mockReturnValueOnce(cancel);
    const handler = vi.fn();
    const { unmount } = renderHook(() => useWailsEvent("test:event", handler));

    unmount();

    expect(cancel).toHaveBeenCalledTimes(1);
  });

  it("cancels the previous subscription when the event name changes", () => {
    const cancelA = vi.fn();
    vi.mocked(EventsOn).mockReturnValueOnce(cancelA);
    const handler = vi.fn();
    const { rerender } = renderHook(({ eventName }) => useWailsEvent(eventName, handler), {
      initialProps: { eventName: "event:a" },
    });

    rerender({ eventName: "event:b" });

    expect(cancelA).toHaveBeenCalledTimes(1);
    expect(EventsOn).toHaveBeenLastCalledWith("event:b", handler);
  });
});
