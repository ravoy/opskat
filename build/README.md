# Desktop build resources

Wails v3 development uses `build/config.yml` and the root `Taskfile.yml`.
`scripts/desktop.mjs` owns binding generation, frontend compilation, Go version
flags and platform packaging. Use `make build` or `make build-embed`.

- `appicon.png` generates the macOS icon; `windows/icon.ico` supplies the Windows icon.
- `darwin/Info.plist` is packaged with `OPSKAT_VERSION` replaced by the build version.
  The bundle identifier remains `com.wails.opskat`. Verification builds use a
  data-directory-specific identifier to isolate native WebView storage.
- `windows/wails.exe.manifest` and generated version metadata produce the `.syso`.
- `windows/installer/*` retain the per-user NSIS installer. Build it with
  `bun scripts/desktop.mjs build --platform windows/amd64 --tags embed_opsctl --nsis`.
  Template expansion occurs in `build/bin/installer`, leaving source resources intact.

The v3 Linux shell requires GTK4 and WebKitGTK 6.0. See the platform matrix and
workflows in `.github/` for CI build dependencies and signing/distribution steps.
