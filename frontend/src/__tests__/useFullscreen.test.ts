import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useFullscreen } from "../hooks/useFullscreen";
import { WindowIsFullscreen } from "@/lib/desktop";

describe("useFullscreen", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns false initially", () => {
    vi.mocked(WindowIsFullscreen).mockResolvedValue(false);
    const { result } = renderHook(() => useFullscreen());
    expect(result.current).toBe(false);
  });

  it("updates to true when WindowIsFullscreen resolves true", async () => {
    vi.mocked(WindowIsFullscreen).mockResolvedValue(true);
    const { result } = renderHook(() => useFullscreen());

    await waitFor(() => {
      expect(result.current).toBe(true);
    });
  });

  it("rechecks on window resize event", async () => {
    vi.mocked(WindowIsFullscreen).mockResolvedValueOnce(false).mockResolvedValueOnce(true);

    const { result } = renderHook(() => useFullscreen());

    // Initial check returns false
    await waitFor(() => {
      expect(WindowIsFullscreen).toHaveBeenCalledTimes(1);
    });

    // Trigger resize
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });

    await waitFor(() => {
      expect(result.current).toBe(true);
    });
  });
});
