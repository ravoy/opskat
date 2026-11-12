import { describe, it, expect, vi, beforeEach } from "vitest";
import { openAssetDefault } from "../lib/openAssetDefault";
import * as asset_entity from "@bindings/github.com/octays/opskat/internal/model/entity/asset_entity/models";

// Mock the dependencies
vi.mock("../lib/assetTypes", () => ({
  getAssetType: vi.fn(),
}));

vi.mock("../lib/openAssetInfoTab", () => ({
  openAssetInfoTab: vi.fn(),
}));

import { getAssetType } from "../lib/assetTypes";
import { openAssetInfoTab } from "../lib/openAssetInfoTab";

function makeAsset(id: number, type: string): asset_entity.Asset {
  return {
    ID: id,
    Name: "test",
    Type: type,
    GroupID: 0,
    Icon: "",
    Tags: "",
    Description: "",
    Config: "",
    CmdPolicy: "",
    SortOrder: 0,
    sshTunnelId: 0,
    Status: 1,
    Createtime: 0,
    Updatetime: 0,
  };
}

describe("openAssetDefault", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls onConnectAsset when asset type canConnect is true", () => {
    const mockOnConnect = vi.fn();
    const asset = makeAsset(1, "ssh");

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    vi.mocked(getAssetType).mockReturnValue({ canConnect: true } as any);

    openAssetDefault(asset, mockOnConnect);

    expect(mockOnConnect).toHaveBeenCalledWith(asset);
    expect(openAssetInfoTab).not.toHaveBeenCalled();
  });

  it("calls openAssetInfoTab when asset type canConnect is false", () => {
    const mockOnConnect = vi.fn();
    const asset = makeAsset(42, "unknown");

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    vi.mocked(getAssetType).mockReturnValue({ canConnect: false } as any);

    openAssetDefault(asset, mockOnConnect);

    expect(openAssetInfoTab).toHaveBeenCalledWith(42);
    expect(mockOnConnect).not.toHaveBeenCalled();
  });

  it("calls openAssetInfoTab when getAssetType returns undefined", () => {
    const mockOnConnect = vi.fn();
    const asset = makeAsset(99, "nonexistent");

    vi.mocked(getAssetType).mockReturnValue(undefined);

    openAssetDefault(asset, mockOnConnect);

    expect(openAssetInfoTab).toHaveBeenCalledWith(99);
    expect(mockOnConnect).not.toHaveBeenCalled();
  });
});
