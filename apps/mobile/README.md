# mobile

A phone showing one vault. The core is the one the window uses: `gomobile` binds it as an Android library, the application starts it on the loopback address, and the page calls it using the schema in `libs/core/protocol`.

## What is here

- `bind/` — the Go side: `Start(dir)` opens the vault under `dir` and returns the port it serves on. `Stop()` closes it.
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

The policy is the `<meta http-equiv>` in `index.html`, checked by `src/App.policy.test.ts` against every address the page requests. The window on the desktop is served its policy as a header; here the package serves the page and sets none, so the page contains its own. This means `frame-ancestors` is not available, because a page cannot set it for itself. It also means the bridge that Capacitor injects (an inline script inserted directly after `<head>`) runs before the policy is parsed, so the policy does not apply to it.

Three things are still to be checked on a device: that the Ionic styles, injected at runtime, are drawn under `style-src 'unsafe-inline'`; that on a WebView too old for a message listener the bridge's reply, evaluated into the page from the native side, is not read as a script the policy refuses; and that every part of the application is shown.
