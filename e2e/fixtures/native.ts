import { expect, type APIRequestContext } from "@playwright/test";

export async function nativeTool(request: APIRequestContext, name: string, args: Record<string, unknown> = {}) {
  const port = Number(process.env.WAILS_MCP_PORT || Number(process.env.OPSKAT_E2E_PORT) + 2000);
  const response = await request.post(`http://127.0.0.1:${port}/mcp`, {
    data: { jsonrpc: "2.0", id: 1, method: "tools/call", params: { name, arguments: args } },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  if (payload.error || payload.result?.isError) throw new Error(JSON.stringify(payload));
  return JSON.parse(payload.result.content.find((item: { type: string }) => item.type === "text").text);
}

