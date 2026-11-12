/**
 * The runtime the desktop shell serves beside its pages. A page served without it (stories, tests)
 * gets no runtime, and what needs one does nothing or falls back to the browser.
 */
import type { FrameRuntime, FrameRuntimeLoader } from '../lib/runtime'

/** The path the desktop shell serves its runtime module at. */
const RUNTIME_PATH = '/wails/runtime.js'

let runtime: Promise<FrameRuntime | undefined> | undefined

// Stryker disable next-line all: dynamic import of /wails/runtime.js is only resolvable in the Wails webview
export const loadFrameRuntime: FrameRuntimeLoader = () => {
  // An address built at run time is not resolved by a bundler of the package that imports this one.
  runtime ??= import(/* @vite-ignore */ new URL(RUNTIME_PATH, window.location.href).href).then(
    (loaded: FrameRuntime) => loaded,
    () => undefined,
  )
  return runtime
}

/**
 * Opens an address in the system browser through the runtime, and in a new browsing context where the
 * page has none. The caller decides which schemes it opens. A failure of the runtime to open the
 * address rejects the returned promise.
 */
export async function openExternalUrl(
  href: string,
  load: FrameRuntimeLoader = loadFrameRuntime,
): Promise<void> {
  const open = (await load())?.Browser?.OpenURL
  if (open) {
    await open(href)
    return
  }
  window.open(href, '_blank')
}
