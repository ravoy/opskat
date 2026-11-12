// Cross-platform e2e runner: runs `playwright test` (forwarding any extra args),
// then cleans up AFTER Playwright has fully exited.
//
// Why a Node wrapper instead of doing cleanup elsewhere:
//   - Not in Playwright's globalTeardown: that runs while `wails3 dev` is still the
//     *managed* webServer, so killing there SIGTERMs the live server (exit 143), and
//     on Windows the still-open opskat.db can't be deleted (EPERM). See harness guide §7.
//   - Not in a Makefile `pkill`: that's Unix-only and self-matches the recipe shell's
//     own command line on Linux (procps reads /proc/<pid>/cmdline), SIGTERMing make.
// Running here means cleanup happens once Playwright has torn down the webServer —
// the app is gone (db closed), vite is orphaned — and it works the same on
// Windows / macOS / Linux, so `make test-e2e` and a bare `pnpm test` both behave.
//
// This wrapper is for the *suite*. The interactive verification sandbox is
// `sandbox.mjs`, which holds the app open instead of running tests.
import { spawn } from "node:child_process";
import { createRequire } from "node:module";
import { rmSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { reapOrphanSuiteApp, reapOrphanVite, suiteDataDir, webserverLogPath } from "./harness/env.js";

const here = dirname(fileURLToPath(import.meta.url)); // e2e/

const require = createRequire(import.meta.url);
const playwrightCli = require.resolve("@playwright/test/cli");

const child = spawn(
  process.execPath,
  [playwrightCli, "test", ...process.argv.slice(2)],
  { cwd: here, stdio: "inherit" },
);

child.on("exit", (code) => {
  void cleanup({ preserveWebserverLog: code !== 0 }).then(() => {
    // Mirror the child's outcome; a signal-killed run (code === null) counts as failure.
    process.exit(code ?? 1);
  });
});

async function cleanup({ preserveWebserverLog }) {
  reapOrphanVite();
  // Playwright's teardown leaves `scripts/desktop.mjs run` and the app alive; reap
  // them by the run's OPSKAT_DATA_DIR, then let go of opskat.db before the wipe
  // (still-open handles EPERM on Windows and linger on macOS).
  const dataDir = suiteDataDir();
  reapOrphanSuiteApp(dataDir);
  const dbPath = join(dataDir, "opskat.db");
  for (let attempt = 0; attempt < 20; attempt++) {
    try {
      rmSync(dbPath, { force: true });
      break;
    } catch {
      await new Promise((resolve) => setTimeout(resolve, 250));
    }
  }
  rmSync(dataDir, { recursive: true, force: true });
  if (!preserveWebserverLog) {
    rmSync(webserverLogPath(), { force: true });
  }
}
