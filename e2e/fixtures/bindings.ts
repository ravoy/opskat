import type { Page } from "@playwright/test";

// Wails v3 没有 v2 注入的 window.go / window.runtime 全局：生成的 bindings 是
// dev server 直接服务的 ES 模块，其调用经 verification transport 注入的桥接
//（/wails/transport.js）路由进原生应用。在页面里动态 import 这些模块，就是
// 浏览器页面对原生窗口 `js_eval` + `import('/src/lib/desktop.ts')`
//（native-windows.spec.ts）的对应做法——specs 借此用真实 binding 准备前置状态。
export async function callBinding<T = unknown>(
  page: Page,
  module: string,
  method: string,
  ...args: unknown[]
): Promise<T> {
  const result = await page.evaluate(
    async ({ module, method, args }) => {
      const mod = (await import(module)) as Record<string, (...a: unknown[]) => Promise<unknown>>;
      return mod[method](...args);
    },
    { module, method, args },
  );
  return result as T;
}

/** 在页面里触发一个 Wails 自定义事件（经 Go 广播回所有窗口），替代 v2 的 window.runtime.EventsEmit。 */
export async function emitWailsEvent(page: Page, name: string, data: unknown): Promise<void> {
  await page.evaluate(
    async ({ name, data }) => {
      // 路径放进变量再动态 import：tsc 不把非字面量说明符当模块解析，浏览器里
      // 仍按 dev server 的 URL 解析。
      const path = "/src/lib/desktop.ts";
      const { EventsEmit } = (await import(path)) as { EventsEmit: (n: string, d?: unknown) => Promise<void> };
      await EventsEmit(name, data);
    },
    { name, data },
  );
}
