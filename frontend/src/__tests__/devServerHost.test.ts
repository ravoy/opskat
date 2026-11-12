import { afterAll, beforeAll, expect, it } from "vitest";
import { createServer, type ViteDevServer } from "vite";
import path from "node:path";
import { request } from "node:http";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";

const cacheDir = mkdtempSync(path.join(tmpdir(), "opskat-vite-host-test-"));
let server: ViteDevServer;
let port: number;

beforeAll(async () => {
  server = await createServer({
    configFile: path.resolve("vite.config.ts"),
    cacheDir,
    server: { port: 0, strictPort: true, host: "127.0.0.1", hmr: false, warmup: { clientFiles: [] } },
  });
  await server.listen();
  const address = server.httpServer!.address();
  if (!address || typeof address === "string") throw new Error("Vite did not open a TCP listener");
  port = address.port;
}, 30_000);

afterAll(async () => {
  await server?.close();
  rmSync(cacheDir, { recursive: true, force: true });
});

function nativeRequest(host: string): Promise<{ status: number; body: string }> {
  return new Promise((resolve, reject) => {
    const req = request({ host: "127.0.0.1", port, path: "/", headers: { Host: host } }, (response) => {
      let body = "";
      response.setEncoding("utf8");
      response.on("data", (chunk) => {
        body += chunk;
      });
      response.on("end", () => resolve({ status: response.statusCode!, body }));
    });
    req.on("error", reject);
    req.end();
  });
}

it("serves the native WebView's persisted wails host without disabling host checks", async () => {
  const response = await nativeRequest("wails");
  expect(response.status).toBe(200);
  expect(response.body).toContain('<div id="root">');
  expect((await nativeRequest("untrusted.example")).status).toBe(403);
});
