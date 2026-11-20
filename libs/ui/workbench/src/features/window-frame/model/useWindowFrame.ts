/**
 * The frame of the desktop window: minimize, maximize, close, and the platform the window runs on.
 * The calls go through the Wails runtime served beside the page; a page served without it (stories,
 * tests) gets a frame that does nothing.
 */
import { computed, getCurrentScope, onScopeDispose, ref, watch, type Ref } from 'vue'
import { doc, hasDocument, runInBackground } from '@numen/ui'
import {
  getWindowEdge as getTopEdge,
  getWindowRadius,
  readMajorVersion,
  WINDOW_EDGE_VARIABLE,
  WINDOW_RADIUS_VARIABLE,
  type WindowPlatform,
} from '../lib/corners'
import { getWindowEdge, type FrameEdge } from './frameEdge'

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

interface FrameControls {
  getPlatform?(): WindowPlatform
  getVersion?(): Promise<string | undefined>
  isMaximized?(): Promise<boolean>
  isFullscreen?(): Promise<boolean>
  minimize?(): Promise<void>
  toggleMaximize?(): Promise<void>
  close?(): Promise<void>
}

const ignoreFailure = (): undefined => undefined

async function readMacVersion(controls: FrameControls | undefined): Promise<number | undefined> {
  try {
    return readMajorVersion(await controls?.getVersion?.())
  } catch {
    return undefined
  }
}

/**
 * The frame of the window this page is drawn in, read through the runtime the shell loads: its platform, its
 * maximized and full screen state, which a resize reads again, and its buttons.
 */
export function useWindowFrame(load: () => Promise<FrameControls | undefined>): WindowFrame {
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
      if (!hasDocument()) return
      doc.documentElement.style.setProperty(WINDOW_RADIUS_VARIABLE, `${radius}px`)
      doc.documentElement.style.setProperty(WINDOW_EDGE_VARIABLE, `${edge}px`)
    },
    { immediate: true },
  )

  /** Reads the maximized and the full screen state, which a resize changes. */
  async function readWindowState(): Promise<void> {
    const controls = await load()
    isMaximized.value = (await controls?.isMaximized?.()) ?? false
    isFullscreen.value = (await controls?.isFullscreen?.()) ?? false
  }

  async function minimizeWindow(): Promise<void> {
    await (await load())?.minimize?.()
  }

  async function toggleMaximizeWindow(): Promise<void> {
    await (await load())?.toggleMaximize?.()
    await readWindowState().catch(ignoreFailure)
  }

  async function closeWindow(): Promise<void> {
    await (await load())?.close?.()
  }

  runInBackground(
    load().then(async (loaded) => {
      platform.value = loaded?.getPlatform?.() ?? 'windows'
      if (platform.value === 'darwin') macVersion.value = await readMacVersion(loaded)
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
