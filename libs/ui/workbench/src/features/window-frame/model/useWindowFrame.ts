/**
 * The frame of the desktop window: minimize, maximize, close, and the platform the window runs on.
 * The calls go through the Wails runtime served beside the page; a page served without it (stories,
 * tests) gets a frame that does nothing.
 */
import { computed, getCurrentScope, onScopeDispose, ref, watch, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import {
  getWindowEdge as getTopEdge,
  getWindowRadius,
  readMajorVersion,
  WINDOW_EDGE_VARIABLE,
  WINDOW_RADIUS_VARIABLE,
  type WindowPlatform,
} from '../lib/corners'
import type { FrameRuntime, FrameRuntimeLoader } from '../lib/runtime'
import { MAC_BUTTONS_INSET } from '../lib/sizes'

/** The room the system buttons take in the top corners of the window, in pixels. */
export interface FrameEdge {
  readonly leftInset: number
  readonly rightInset: number
}

export interface WindowFrame {
  readonly platform: Readonly<Ref<WindowPlatform>>
  readonly isMaximized: Readonly<Ref<boolean>>
  readonly isFullscreen: Readonly<Ref<boolean>>
  readonly windowEdge: Readonly<Ref<FrameEdge>>
  readonly windowRadius: Readonly<Ref<number>>
  minimizeWindow(): Promise<void>
  toggleMaximizeWindow(): Promise<void>
  closeWindow(): Promise<void>
}

function getPlatform(system: FrameRuntime['System']): WindowPlatform {
  if (system?.IsMac?.()) return 'darwin'
  return system?.IsLinux?.() ? 'linux' : 'windows'
}

/**
 * The room the system buttons take in the top corners: the macOS buttons at the left, except in full
 * screen, where macOS shows none. Windows and Linux have no buttons without a title bar.
 */
export function getWindowEdge(platform: WindowPlatform, isFullscreen: boolean): FrameEdge {
  const hasMacButtons = platform === 'darwin' && !isFullscreen
  return { leftInset: hasMacButtons ? MAC_BUTTONS_INSET : 0, rightInset: 0 }
}

const ignoreFailure = (): undefined => undefined

async function readMacVersion(system: FrameRuntime['System']): Promise<number | undefined> {
  try {
    return readMajorVersion((await system?.Environment?.())?.OSInfo?.Version)
  } catch {
    return undefined
  }
}

/**
 * The frame of the window this page is drawn in, read through the runtime the shell loads: its platform, its
 * maximized and full screen state, which a resize reads again, and its buttons.
 */
export function useWindowFrame(load: FrameRuntimeLoader): WindowFrame {
  const platform = ref<WindowPlatform>('windows')
  const isMaximized = ref(false)
  const isFullscreen = ref(false)
  const macVersion = ref<number | undefined>(undefined)
  const windowRadius = computed(() =>
    getWindowRadius(platform.value, macVersion.value, isFullscreen.value),
  )
  watch(
    [windowRadius, () => getTopEdge(platform.value, isFullscreen.value)],
    ([radius, edge]) => {
      if (typeof document === 'undefined') return
      document.documentElement.style.setProperty(WINDOW_RADIUS_VARIABLE, `${radius}px`)
      document.documentElement.style.setProperty(WINDOW_EDGE_VARIABLE, `${edge}px`)
    },
    { immediate: true },
  )

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
      if (platform.value === 'darwin') macVersion.value = await readMacVersion(loaded?.System)
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
    windowRadius,
    windowEdge: computed(() => getWindowEdge(platform.value, isFullscreen.value)),
    minimizeWindow,
    toggleMaximizeWindow,
    closeWindow,
  }
}
