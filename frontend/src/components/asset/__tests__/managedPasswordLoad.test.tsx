import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createRef } from "react";
import { EtcdConfigSection } from "@/components/asset/EtcdConfigSection";
import type { AssetFormContext } from "@/lib/assetTypes/formContract";
import * as asset_entity from "@bindings/github.com/octays/opskat/internal/model/entity/asset_entity/models";
import { ListCredentialsByType } from "@clients/system";

const ctx: AssetFormContext = { isEdit: false, encryptPassword: async (p) => p };

describe("managed password load failure in an asset config section", () => {
  beforeEach(() => {
    vi.mocked(ListCredentialsByType).mockReset();
  });

  it("shows the reason and retry, not the 'none' hint, and retry recovers", async () => {
    vi.mocked(ListCredentialsByType).mockRejectedValueOnce(new Error("vault locked"));
    render(
      <EtcdConfigSection
        ref={createRef()}
        editAsset={new asset_entity.Asset({ Type: "etcd", Config: '{"endpoints":["a:2379"]}' })}
        ctx={ctx}
        onValidityChange={() => {}}
      />
    );
    await userEvent.click(screen.getByTestId("password-source-managed"));
    expect(await screen.findByText(/vault locked/)).toBeInTheDocument();
    expect(screen.queryByText("asset.noManagedPasswords")).not.toBeInTheDocument();

    vi.mocked(ListCredentialsByType).mockResolvedValueOnce([]);
    await userEvent.click(screen.getByTestId("async-state-retry"));
    expect(await screen.findByText("asset.noManagedPasswords")).toBeInTheDocument();
  });
});
