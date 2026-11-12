import { toast } from "sonner";
import { OpenPageWindow } from "@clients/system";
import { PageWindowRequest } from "@bindings/github.com/octays/opskat/internal/desktop/models";
import { errorMessage } from "./errorMessage";

export async function openPageWindow(options: Partial<PageWindowRequest> & { page: string }): Promise<void> {
  try {
    await OpenPageWindow(new PageWindowRequest(options));
  } catch (error) {
    toast.error(errorMessage(error));
  }
}

export function currentWindowPage(): string | null {
  return new URLSearchParams(window.location.search).get("page");
}
