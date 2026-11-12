import { describe, expect, it } from "vitest";
import { createIPCClient } from "./ipcClient";

describe("IPC result contracts", () => {
  it("preserves cancellation while rejecting missing required results", async () => {
    const client = createIPCClient(
      {
        getAsset: async () => null as { id: number } | null,
        selectFile: async () => null as string | null,
        list: async () => [null] as ({ id: number } | null)[],
        save: async () => {},
      },
      ["selectFile"]
    );
    await expect(client.selectFile()).resolves.toBeNull();
    await expect(client.getAsset()).rejects.toThrow("getAsset: backend returned no result");
    await expect(client.list()).rejects.toThrow("null list item");
    await expect(client.save()).resolves.toBeUndefined();
  });
});
