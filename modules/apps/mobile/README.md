# mobile

A phone showing one vault. The core is the one the window uses: `gomobile`
binds it as an Android library, the application starts it on the loopback, and
the page asks it over the schema in `modules/libs/protocol`.

## What is here

- `bind/` — the Go side: `Start(dir)` opens the vault under `dir` and answers
  with the port it serves on. `Stop()` closes it.
- `android/app/src/main/java/md/numen/spike/NumenCorePlugin.java` — the
  Capacitor plugin. It hands the core the folder the platform gave the
  application and passes the port to the page.
- `src/core.ts` — the page's half: the port, then a generated client.
- `src/App.vue` — what the vault holds, and one note more.

## Building

The toolchain: a JDK 21, an Android SDK holding build-tools 35 and an NDK, and
`gomobile` on `PATH`.

```
export ANDROID_SDK_ROOT=…      # holds ndk/ and build-tools/35.0.0
export JAVA_HOME=…             # a JDK 21
make run                       # binds, builds, installs, opens
```

## What runs on a phone and what does not

The whole of `modules/libs/core` compiles for `android/arm64` with the NDK, OCR
included. Nothing is cut for this build.

`modernc.org/libc` on `android/amd64` — an emulator — calls `lstat`, `open`,
`unlink` and their kin directly, and Android's seccomp filter kills a process
that does. On `arm64` the same library goes through the `*at` forms and is
fine, so a phone is unaffected. `go.work` at the root of this tree points at a
patched copy for the emulator; a build for a phone wants neither.
