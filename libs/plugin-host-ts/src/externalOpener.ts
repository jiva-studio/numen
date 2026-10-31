/**
 * Opening an address in the system browser.
 *
 * The window has no address bar. The Wails runtime served beside the page opens an address in the
 * system browser; a page served without it opens a new browsing context. A plugin's `openExternal`
 * passes an http or https address to this and fails any other address before the window sees it.
 */
import { asFailure, asValue, formatErrorMessage, type Result } from '@numen/plugin-sdk'

/** The path the window serves its Wails runtime module at. */
const RUNTIME_PATH = '/wails/runtime.js'

/** The part of the Wails runtime module that opens an address. */
export interface BrowserRuntime {
  readonly Browser?: { readonly OpenURL?: (href: string) => unknown }
}

/** Loads the runtime module; resolves to undefined where the page is not served by Wails. */
export type RuntimeLoader = () => Promise<BrowserRuntime | undefined>

/** Opens an address in the system browser. */
export type ExternalOpener = (url: string) => Promise<void>

let runtime: Promise<BrowserRuntime | undefined> | undefined

/** Loads the Wails runtime module once. */
const loadRuntime: RuntimeLoader = () => {
  runtime ??= import(/* @vite-ignore */ RUNTIME_PATH).then(
    (loaded: BrowserRuntime) => loaded,
    () => undefined,
  )
  return runtime
}

/**
 * Opens an address in the system browser. The caller decides which schemes it opens. A failure of the
 * runtime to open the address rejects the returned promise.
 */
export async function openExternalUrl(
  href: string,
  load: RuntimeLoader = loadRuntime,
): Promise<void> {
  const open = (await load())?.Browser?.OpenURL
  if (open) {
    await open(href)
    return
  }
  window.open(href, '_blank')
}

/** Whether an address is an absolute http or https address. */
export function isWebAddress(url: string): boolean {
  try {
    const { protocol } = new URL(url)
    return protocol === 'http:' || protocol === 'https:'
  } catch {
    return false
  }
}

export function createExternalOpener(open: ExternalOpener): (url: string) => Promise<Result<null>> {
  return async (url) => {
    if (!isWebAddress(url)) return asFailure(`not a web address: ${url}`)
    try {
      await open(url)
      return asValue(null)
    } catch (error) {
      return asFailure(formatErrorMessage(error))
    }
  }
}
