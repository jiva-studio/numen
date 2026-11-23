/** The part of the runtime of the desktop shell the pages use, under the names the shell gives it. */
interface FrameRuntime {
  readonly Window?: {
    readonly Focus?: () => Promise<void>
    readonly Minimise?: () => Promise<void>
    readonly ToggleMaximise?: () => Promise<void>
    readonly Close?: () => Promise<void>
    readonly IsMaximised?: () => Promise<boolean>
    readonly IsFullscreen?: () => Promise<boolean>
  }
  readonly System?: {
    readonly Environment?: () => Promise<{
      readonly OS: string
      readonly OSInfo?: { readonly Version?: string }
    }>
  }
  readonly Browser?: { readonly OpenURL?: (href: string) => void | Promise<void> }
}

/** Loads the runtime module; resolves to undefined where the page is not served by the desktop shell. */
export type FrameRuntimeLoader = () => Promise<FrameRuntime | undefined>

const RUNTIME_PATH = '/wails/runtime.js'

/** A loader that imports the runtime module once and gives the same promise on every call. */
export function createFrameRuntimeLoader(): FrameRuntimeLoader {
  let runtime: Promise<FrameRuntime | undefined> | undefined
  // Stryker disable next-line all: dynamic import of /wails/runtime.js is only resolvable in the Wails webview
  return () => {
    // A path built at run time is left to the webview, not resolved by the bundler.
    runtime ??= import(/* @vite-ignore */ new URL(RUNTIME_PATH, window.location.href).href).then(
      (loaded: FrameRuntime) => loaded,
      () => undefined,
    )
    return runtime
  }
}

/**
 * Opens an address in the system browser through the runtime, and in a new browsing context where the
 * page has none. The caller decides which schemes it opens. A failure of the runtime to open the
 * address rejects the returned promise.
 */
export async function openExternalUrl(href: string, load: FrameRuntimeLoader): Promise<void> {
  const open = (await load())?.Browser?.OpenURL
  if (open) {
    await open(href)
    return
  }
  window.open(href, '_blank')
}

/** Adapts the shell controls and platform queries used by the window chrome. */
export function createWindowFrameControls(load: FrameRuntimeLoader) {
  const runtime = load()
  return {
    async getEnvironment() {
      const environment = await (await runtime)?.System?.Environment?.()
      const os = environment?.OS
      const platform: 'darwin' | 'linux' | 'windows' =
        os === 'darwin' || os === 'linux' ? os : 'windows'
      return { platform, version: environment?.OSInfo?.Version }
    },
    isMaximized: async () => (await (await runtime)?.Window?.IsMaximised?.()) ?? false,
    isFullscreen: async () => (await (await runtime)?.Window?.IsFullscreen?.()) ?? false,
    minimize: async () => {
      await (await runtime)?.Window?.Minimise?.()
    },
    toggleMaximize: async () => {
      await (await runtime)?.Window?.ToggleMaximise?.()
    },
    close: async () => {
      await (await runtime)?.Window?.Close?.()
    },
  }
}
