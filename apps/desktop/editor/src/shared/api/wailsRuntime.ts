/**
 * The Wails runtime the window serves beside its pages. A page served without it (stories, tests)
 * gets no runtime, and what needs one does nothing or falls back to the browser.
 */

/** The part of the Wails runtime module the pages use, under the names Wails gives it. */
interface WailsRuntime {
  readonly Window?: {
    readonly Focus?: () => Promise<void>
    readonly Minimise?: () => Promise<void>
    readonly ToggleMaximise?: () => Promise<void>
    readonly Close?: () => Promise<void>
    readonly IsMaximised?: () => Promise<boolean>
    readonly IsFullscreen?: () => Promise<boolean>
  }
  readonly System?: {
    readonly IsMac?: () => boolean
    readonly IsWindows?: () => boolean
    readonly IsLinux?: () => boolean
    readonly Environment?: () => Promise<{ readonly OSInfo?: { readonly Version?: string } }>
  }
  readonly Browser?: { readonly OpenURL?: (href: string) => void | Promise<void> }
}

/** Loads the runtime module; resolves to undefined where the page is not served by Wails. */
export type WailsRuntimeLoader = () => Promise<WailsRuntime | undefined>

/** The path the window serves its Wails runtime module at. */
const RUNTIME_PATH = '/wails/runtime.js'

let runtime: Promise<WailsRuntime | undefined> | undefined

// Stryker disable next-line all: dynamic import of /wails/runtime.js is only resolvable in the Wails webview
export const loadWailsRuntime: WailsRuntimeLoader = () => {
  // An address built at run time is not resolved by a bundler of the package that imports this one.
  runtime ??= import(/* @vite-ignore */ new URL(RUNTIME_PATH, window.location.href).href).then(
    (loaded: WailsRuntime) => loaded,
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
  load: WailsRuntimeLoader = loadWailsRuntime,
): Promise<void> {
  const open = (await load())?.Browser?.OpenURL
  if (open) {
    await open(href)
    return
  }
  window.open(href, '_blank')
}
