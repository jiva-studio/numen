/**
 * The frame of the desktop window: minimize, maximize, close, and the platform the window runs on.
 * The calls go through the Wails runtime served beside the page; a page served without it (stories,
 * tests) gets a frame that does nothing.
 */
import { computed, getCurrentScope, onScopeDispose, ref, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { MAC_BUTTONS_INSET, type WindowEdge } from '@numen/shell'

/** The operating system of the window, as `runtime.GOOS` names it. */
export type WindowPlatform = 'darwin' | 'windows' | 'linux'

/** The path the window serves its Wails runtime module at. */
const RUNTIME_PATH = '/wails/runtime.js'

/** The part of the Wails runtime module the frame uses, under the names Wails gives it. */
export interface FrameRuntime {
  readonly Window?: {
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
  }
}

/** Loads the runtime module; resolves to undefined where the page is not served by Wails. */
export type FrameRuntimeLoader = () => Promise<FrameRuntime | undefined>

export interface WindowFrame {
  readonly platform: Readonly<Ref<WindowPlatform>>
  readonly isMaximized: Readonly<Ref<boolean>>
  /** True while the window fills the screen; macOS then shows no system buttons. */
  readonly isFullscreen: Readonly<Ref<boolean>>
  /** The room the system buttons take in the top corners while no title bar is drawn. */
  readonly windowEdge: Readonly<Ref<WindowEdge>>
  minimizeWindow(): Promise<void>
  toggleMaximizeWindow(): Promise<void>
  closeWindow(): Promise<void>
}

let runtime: Promise<FrameRuntime | undefined> | undefined

// Stryker disable next-line all: dynamic import of /wails/runtime.js is only resolvable in the Wails webview
const loadRuntime: FrameRuntimeLoader = () => {
  runtime ??= import(/* @vite-ignore */ RUNTIME_PATH).then(
    (loaded: FrameRuntime) => loaded,
    () => undefined,
  )
  return runtime
}

/** The platform the runtime reports; Windows where the page is served without the runtime. */
function getPlatform(system: FrameRuntime['System']): WindowPlatform {
  if (system?.IsMac?.()) return 'darwin'
  return system?.IsLinux?.() ? 'linux' : 'windows'
}

/**
 * The room the system buttons take in the top corners: the macOS buttons at the left, except in full
 * screen, where macOS shows none. Windows and Linux have no buttons without a title bar.
 */
export function getWindowEdge(platform: WindowPlatform, isFullscreen: boolean): WindowEdge {
  const hasMacButtons = platform === 'darwin' && !isFullscreen
  return { leftInset: hasMacButtons ? MAC_BUTTONS_INSET : 0, rightInset: 0 }
}

/** A read of the window state that fails leaves the state as it was. */
const ignoreFailure = (): undefined => undefined

export function useWindowFrame(load: FrameRuntimeLoader = loadRuntime): WindowFrame {
  const platform = ref<WindowPlatform>('windows')
  const isMaximized = ref(false)
  const isFullscreen = ref(false)

  /** Reads the maximized and the full screen state, which a resize changes. */
  async function readWindowState(): Promise<void> {
    const wailsWindow = (await load())?.Window
    isMaximized.value = (await wailsWindow?.IsMaximised?.()) ?? false
    isFullscreen.value = (await wailsWindow?.IsFullscreen?.()) ?? false
  }

  async function minimizeWindow(): Promise<void> {
    await (await load())?.Window?.Minimise?.()
  }

  async function toggleMaximizeWindow(): Promise<void> {
    await (await load())?.Window?.ToggleMaximise?.()
    await readWindowState().catch(ignoreFailure)
  }

  async function closeWindow(): Promise<void> {
    await (await load())?.Window?.Close?.()
  }

  runInBackground(
    load().then(async (loaded) => {
      platform.value = getPlatform(loaded?.System)
      await readWindowState()
    }),
    ignoreFailure,
  )

  // A resize is how a window gets maximized or restored by the system, not only by the button.
  const onResize = () => runInBackground(readWindowState(), ignoreFailure)
  globalThis.addEventListener('resize', onResize)
  if (getCurrentScope()) onScopeDispose(() => globalThis.removeEventListener('resize', onResize))

  return {
    platform,
    isMaximized,
    isFullscreen,
    windowEdge: computed(() => getWindowEdge(platform.value, isFullscreen.value)),
    minimizeWindow,
    toggleMaximizeWindow,
    closeWindow,
  }
}
