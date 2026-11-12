import type * as asset_entity from "@bindings/github.com/octays/opskat/internal/model/entity/asset_entity/models";
import { getAssetType } from "./assetTypes";
import { openAssetInfoTab } from "./openAssetInfoTab";

export function openAssetDefault(asset: asset_entity.Asset, onConnectAsset: (asset: asset_entity.Asset) => void): void {
  const def = getAssetType(asset.Type);
  if (def?.canConnect) {
    onConnectAsset(asset);
  } else {
    openAssetInfoTab(asset.ID);
  }
}
