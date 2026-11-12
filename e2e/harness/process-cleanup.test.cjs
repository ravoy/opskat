// Run with: bun e2e/harness/process-cleanup.test.cjs
// Exercise cleanup against real child processes in a synthetic checkout, so the
// regression cannot terminate tests running in the actual working directory.
const assert = require("node:assert/strict");
const { spawn } = require("node:child_process");
const { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync } = require("node:fs");
const { tmpdir } = require("node:os");
const { join } = require("node:path");
const Module = require("node:module");
const { setTimeout: delay } = require("node:timers/promises");

function loadHarness(root) {
  const filename = join(root, "e2e", "harness", "env.js");
  const harness = new Module(filename);
  harness.filename = filename;
  harness.paths = module.paths;
  harness._compile(readFileSync(join(__dirname, "env.js"), "utf8"), filename);
  return harness.exports;
}

if (process.platform === "win32") {
  console.log("Unix process-cleanup regression: skipped on Windows");
} else {
  void (async () => {
    const root = mkdtempSync(join(tmpdir(), "opskat-cleanup-[x]+-"));
    const children = [];
    try {
      for (const tool of ["vite", "vitest"]) {
        const script = join(root, "frontend", "node_modules", tool, "bin", `${tool}.js`);
        mkdirSync(join(script, ".."), { recursive: true });
        writeFileSync(script, 'setInterval(() => {}, 1000); console.log("ready");');
        const child = spawn(process.execPath, [script], { stdio: ["ignore", "pipe", "inherit"] });
        children.push(child);
        await new Promise((resolve, reject) => {
          child.stdout.once("data", resolve);
          child.once("error", reject);
          child.once("exit", (code) => reject(new Error(`${tool} exited before ready: ${code}`)));
        });
      }
      const harness = loadHarness(root);
      harness.reapOrphanVite();
      await delay(200);
      assert.notEqual(children[0].signalCode, null, "orphan Vite server must be reaped");
      assert.equal(children[1].signalCode, null, "cleanup must leave Vitest running");
      console.log("Process cleanup preserves Vitest and reaps Vite");

      // The suite-app reaper targets the run's OPSKAT_DATA_DIR: a process carrying
      // it dies, one with a different data dir (a developer's `make dev`, another
      // workspace) survives.
      const keepDir = mkdtempSync(join(tmpdir(), "opskat-cleanup-keep-"));
      const reapDir = mkdtempSync(join(tmpdir(), "opskat-cleanup-reap-"));
      const hold = (dataDir) =>
        spawn(process.execPath, ["-e", "setInterval(() => {}, 1000)"], {
          stdio: "ignore",
          env: { ...process.env, OPSKAT_DATA_DIR: dataDir },
        });
      const survivor = hold(keepDir);
      const victim = hold(reapDir);
      await delay(100);
      assert.equal(victim.exitCode, null, "victim must be alive before cleanup");
      harness.reapOrphanSuiteApp(reapDir);
      await delay(300);
      assert.equal(victim.signalCode, "SIGTERM", "process carrying the reaped data dir must be SIGTERMed");
      assert.equal(survivor.exitCode, null, "process with another data dir must survive");
      assert.equal(survivor.signalCode, null, "survivor must not have been signalled");
      console.log("Suite-app cleanup reaps only the run's own data dir");
      survivor.kill("SIGTERM");
    } finally {
      for (const child of children) {
        if (child.exitCode === null && child.signalCode === null) child.kill("SIGTERM");
      }
      rmSync(root, { recursive: true, force: true });
    }
  })().catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
}
