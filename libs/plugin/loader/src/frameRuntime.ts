/** The part of the runtime of the desktop shell the pages use, under the names the shell gives it. */
export interface FrameRuntime {
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

/** Loads the runtime module; resolves to undefined where the page is not served by the desktop shell. */
export type FrameRuntimeLoader = () => Promise<FrameRuntime | undefined>

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

/** Adapts the shell controls and platform queries used by the window chrome. */
export async function loadWindowFrameControls(load: FrameRuntimeLoader = loadFrameRuntime) {
  const runtime = await load()
  if (!runtime) return undefined
  const { Window: shell, System: system } = runtime
  return {
    getPlatform: (): 'darwin' | 'linux' | 'windows' => {
      if (system?.IsMac?.()) return 'darwin'
      return system?.IsLinux?.() ? 'linux' : 'windows'
    },
    getVersion: async () => (await system?.Environment?.())?.OSInfo?.Version,
    isMaximized: async () => (await shell?.IsMaximised?.()) ?? false,
    isFullscreen: async () => (await shell?.IsFullscreen?.()) ?? false,
    minimize: async () => {
      await shell?.Minimise?.()
    },
    toggleMaximize: async () => {
      await shell?.ToggleMaximise?.()
    },
    close: async () => {
      await shell?.Close?.()
    },
  }
}
