import { createHash } from "node:crypto";
import { spawnSync, spawn } from "node:child_process";
import { readFileSync, writeFileSync, mkdirSync, cpSync, existsSync, unlinkSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
process.chdir(root);
const [command = "build", ...args] = process.argv.slice(2);
const option = (name, fallback) => {
  const index = args.indexOf(name);
  return index === -1 ? fallback : args[index + 1];
};
const dev = args.includes("--dev");
const target = option("--platform", `${process.env.GOOS || (process.platform === "win32" ? "windows" : process.platform)}/${process.env.GOARCH || (process.arch === "x64" ? "amd64" : process.arch)}`);
const [goos, goarch] = target.split("/");
const version = process.env.VERSION || "1.16.0";
const versionMatch = /^v?(\d+\.\d+\.\d+)(?:[-+].*)?$/.exec(version);
if (!versionMatch) throw new Error(`Expected a semantic release version, received: ${version}`);
const numericVersion = versionMatch[1];
const commit = process.env.COMMIT_ID || spawnSync("git", ["rev-parse", "--short", "HEAD"], { encoding: "utf8" }).stdout.trim();
const tags = [dev ? "" : "production", process.env.OPSKAT_E2E === "1" ? "opskat_e2e" : "", option("--tags", process.env.EXTRA_TAGS || "")].filter(Boolean).join(",");
const executable = `opskat${goos === "windows" ? ".exe" : ""}`;
const binary = join("build/bin", executable);
const env = { ...process.env, GOOS: goos, GOARCH: goarch };
if (goos === "darwin") {
  env.CGO_CFLAGS = `${env.CGO_CFLAGS || ""} -mmacosx-version-min=12.0`;
  env.CGO_LDFLAGS = `${env.CGO_LDFLAGS || ""} -mmacosx-version-min=12.0`;
  env.MACOSX_DEPLOYMENT_TARGET = "12.0";
}
function run(program, parameters, options = {}) {
  const result = spawnSync(program, parameters, { stdio: "inherit", env, ...options });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${program} exited with status ${result.status ?? 1}`);
}
function frontend(parameters) {
  run("bun", ["--bun", ...parameters], { cwd: join(root, "frontend") });
}
function bindings() {
  mkdirSync("frontend/dist", { recursive: true });
  writeFileSync("frontend/dist/.keep", "");
  run("wails3", ["generate", "bindings", "-ts", "-d", "frontend/bindings", "."]);
  run(process.execPath, ["scripts/generate-clients.mjs"], { cwd: join(root, "frontend") });
}
if (command === "bindings") {
  bindings();
} else if (command === "frontend") {
  frontend(["run", "dev", "--port", process.env.WAILS_VITE_PORT || "9245", "--strictPort"]);
} else if (command === "run") {
  // Vite may still be starting when the watcher starts its primary process.
  if (process.env.FRONTEND_DEVSERVER_URL) {
    const deadline = Date.now() + 60_000;
    while (true) {
      try {
        const response = await fetch(process.env.FRONTEND_DEVSERVER_URL);
        if (response.ok) break;
      } catch (error) {
        if (Date.now() >= deadline) throw error;
      }
      if (Date.now() >= deadline) throw new Error("Frontend did not become ready");
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
  }
  const launchBinary = goos === "darwin" ? "build/bin/opskat.app/Contents/MacOS/opskat" : binary;
  const child = spawn(resolve(launchBinary), [], { stdio: "inherit", env });
  for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => child.kill(signal));
  child.on("error", (error) => { throw error; });
  child.on("exit", (code) => process.exit(code ?? 1));
} else if (command === "build") {
  bindings();
  frontend(["run", "build"]);
  mkdirSync("build/bin", { recursive: true });
  const ldflags = option("--ldflags", `-s -w -X github.com/cago-frame/cago/configs.Version=${version} -X github.com/octays/opskat/internal/buildinfo.CommitID=${commit}`);
  const buildArgs = ["build", "-buildvcs=false", "-ldflags", `${ldflags}${goos === "windows" && !dev ? " -H windowsgui" : ""}`, "-o", binary];
  if (tags) buildArgs.push("-tags", tags);
  let syso;
  try {
    if (goos === "windows") {
      syso = `wails_windows_${goarch}.syso`;
      const infoPath = "build/bin/windows-info.json";
      writeFileSync(infoPath, readFileSync("build/windows/info.json", "utf8").replaceAll("OPSKAT_VERSION", version).replaceAll("OPSKAT_NUMERIC_VERSION", numericVersion));
      run("wails3", ["generate", "syso", "-arch", goarch, "-icon", "build/windows/icon.ico", "-manifest", "build/windows/wails.exe.manifest", "-info", infoPath, "-out", syso]);
    }
    run("go", buildArgs);
    if (goos === "darwin") {
      const contents = "build/bin/opskat.app/Contents";
      mkdirSync(`${contents}/MacOS`, { recursive: true });
      mkdirSync(`${contents}/Resources`, { recursive: true });
      cpSync(binary, `${contents}/MacOS/opskat`);
      let plist = readFileSync("build/darwin/Info.plist", "utf8").replaceAll("OPSKAT_VERSION", numericVersion);
      if (process.env.OPSKAT_E2E === "1") {
        const identity = createHash("sha256").update(process.env.OPSKAT_DATA_DIR).digest("hex").slice(0, 12);
        plist = plist.replace("com.wails.opskat", `com.wails.opskat.e2e.${identity}`);
      }
      writeFileSync(`${contents}/Info.plist`, plist);
      if (!existsSync("build/darwin/icons.icns")) run("wails3", ["generate", "icons", "-input", "build/appicon.png", "-macfilename", "build/darwin/icons.icns", "-windowsfilename", "build/bin/icon.ico"]);
      cpSync("build/darwin/icons.icns", `${contents}/Resources/iconfile.icns`);
      run("codesign", ["--force", "--deep", "--sign", "-", "build/bin/opskat.app"]);
    }
    if (args.includes("--nsis")) {
      // Keep the existing per-user installer and its portable-mode semantics.
      const staging = "build/bin/installer";
      mkdirSync(staging, { recursive: true });
      const substitutions = { "{{.Name}}": "opskat", "{{.Info.CompanyName}}": "OpsKat", "{{.Info.ProductName}}": "OpsKat", "{{.Info.ProductVersion}}": version, "{{.Info.Copyright}}": "OpsKat contributors" };
      for (const file of ["project.nsi", "wails_tools.nsh"]) {
        let text = readFileSync(`build/windows/installer/${file}`, "utf8");
        text = text.replace(/\{\{range \.Info\.(?:FileAssociations|Protocols)\}\}[\s\S]*?\{\{end\}\}/g, "");
        for (const [token, value] of Object.entries(substitutions)) text = text.replaceAll(token, value);
        text = text.replaceAll('"..\\icon.ico"', '"icon.ico"').replace('OutFile "..\\..\\bin\\', 'OutFile "..\\');
        text = text.replace(/(VI(?:Product|File)Version)\s+"\$\{INFO_PRODUCTVERSION\}\.0"/g, `$1 "${numericVersion}.0"`);
        writeFileSync(`${staging}/${file}`, text);
      }
      cpSync("build/windows/icon.ico", `${staging}/icon.ico`);
      run("wails3", ["generate", "webview2bootstrapper", "-dir", `${staging}/tmp`]);
      run("makensis", [`-DARG_WAILS_${goarch.toUpperCase()}_BINARY=${resolve(binary)}`, "project.nsi"], { cwd: resolve(staging) });
    }
  } finally {
    if (syso && existsSync(syso)) unlinkSync(syso);
  }
} else {
  throw new Error(`Unknown desktop command: ${command}`);
}
