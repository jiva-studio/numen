# mobile

A phone showing one vault. The core is the one the window uses: `gomobile` binds it as an Android library, the application starts it on the loopback address, and the page calls it using the schema in `libs/shared/protocol`.

## What is here

- `bind/` — the Go side: `Start(dir)` opens the vault under `dir` and returns the port it serves on. `Stop()` closes it.
- `cmd/numen-linked/` — the core with Go plugins linked into its binary and served in its process, as the phone will run them. It runs on a computer; its test calls a linked plugin on the route the window uses.
- `android/app/src/main/java/md/numen/spike/NumenCorePlugin.java` — the Capacitor plugin. It passes the core the folder the platform gave the application and passes the port to the page.
- `src/core.ts` — the page's side: it reads the port, then creates a generated client.
- `src/App.vue` — the vault the core opened, and the note it shows first.

## Building

The toolchain: a JDK 21, an Android SDK holding build-tools 35 and an NDK, and `gomobile` on `PATH`.

```
export ANDROID_SDK_ROOT=…      # holds ndk/ and build-tools/35.0.0
export JAVA_HOME=…             # a JDK 21
make run                       # binds, builds, installs, opens
```

## What runs on a phone and what does not

The whole of `libs/core/kernel` compiles for `android/arm64` with the NDK, OCR included. Nothing is cut for this build.

`modernc.org/libc` on `android/amd64` — an emulator — calls `lstat`, `open`, `unlink` and similar system calls directly, and Android's seccomp filter kills a process that does. On `arm64` the same library goes through the `*at` forms and is fine, so a phone is not affected. `go.work` at the root of this tree points at a patched copy for the emulator; a build for a phone needs neither.

## What the page may load

The policy is the `<meta http-equiv>` in `index.html`, pinned by the check `ts/window-sources` (`tools/checks`), and `src/app/App.policy.test.ts` runs what the page asks for. The window on the desktop is served its policy as a header; here the package serves the page and sets none, so the page contains its own. This means `frame-ancestors` is not available, because a page cannot set it for itself. It also means the bridge that Capacitor injects (an inline script inserted directly after `<head>`) runs before the policy is parsed, so the policy does not apply to it.

Three things are still to be checked on a device: that the Ionic styles, injected at runtime, are drawn under `style-src 'unsafe-inline'`; that on a WebView too old for a message listener the bridge's reply, evaluated into the page from the native side, is not read as a script the policy refuses; and that every part of the application is shown.

## Sync on the phone

The phone syncs through the cloud only, with the same protocol and the same `tools-sync` code as the computers, while the application is on the screen. When it leaves the screen it sends what the person wrote within 20 seconds and then stops; when it returns it runs one pass at once and opens the stream again. There is no foreground service, no periodic job and no push channel.

A file of a synced vault that is not a note and not an image of 5 MiB or less is listed with the text "Not downloaded" and is downloaded when the person opens it. Removing a download frees the space and keeps the file in the vault.

### Checks

`scripts/device-check.sh` runs on mridanga against the arm64 emulator `numen-arm64` and is not part of CI. Its steps are written as functions and are still to be filled in.

The list below is run by hand on a real device before the release. Each line says what to do and what must happen.

1. The key vault survives a restart and a device lock: connect, force-stop the application, lock the device, open the application. The identity is still there and no recovery words are asked.
2. Doze and App Standby: leave the application for 30 minutes, open it. One pass runs and the tree is current.
3. Battery: leave the application in the pocket for an hour. The Battery usage screen shows no background use by the application.
4. Mobile data and Wi-Fi: open a file of 100 MB on mobile data, switch on airplane mode in the middle, switch it off, open the file again. The download resumes where it stopped and the file opens.
5. Rotation does not leave the screen: rotate while a download runs. The download goes on and the stream stays open.
6. Low storage: fill the device, open a file larger than the room left. The phone warns and no part of the file is in the vault.
7. Camera link (#2575): scan the code of a computer. The phone shows the synced vault.

The iOS list is this list, to be run once an iOS project exists.
