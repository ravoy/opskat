import { toast } from "sonner";
import { OpenWorkspaceWindow } from "@clients/system";
import { errorMessage } from "./errorMessage";

export async function openWorkspace(assetId = 0): Promise<void> {
  try {
    await OpenWorkspaceWindow(assetId);
  } catch (error) {
    toast.error(errorMessage(error));
  }
}
