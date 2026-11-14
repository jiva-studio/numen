/**
 * The Wails runtime the desktop shell serves beside the page. A page served without it (stories,
 * tests) gets no runtime.
 */
import type { FrameRuntime, FrameRuntimeLoader } from '@numen/workbench'

const RUNTIME_PATH = '/wails/runtime.js'

let runtime: Promise<FrameRuntime | undefined> | undefined

// Stryker disable next-line all: dynamic import of /wails/runtime.js is only resolvable in the Wails webview
export const loadFrameRuntime: FrameRuntimeLoader = () => {
  // A path built at run time is left to the webview, not resolved by the bundler.
  runtime ??= import(/* @vite-ignore */ new URL(RUNTIME_PATH, window.location.href).href).then(
    (loaded: FrameRuntime) => loaded,
    () => undefined,
  )
  return runtime
}
