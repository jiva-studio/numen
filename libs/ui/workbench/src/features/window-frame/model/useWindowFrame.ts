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
  getEnvironment(): Promise<{ platform: WindowPlatform; version: string | undefined }>
  isMaximized(): Promise<boolean>
  isFullscreen(): Promise<boolean>
  minimize(): Promise<void>
  toggleMaximize(): Promise<void>
  close(): Promise<void>
}

const ignoreFailure = (): undefined => undefined

/** Reads the platform once at startup and updates frame geometry on resize. */
export function useWindowFrame(controls: FrameControls): WindowFrame {
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
    isMaximized.value = await controls.isMaximized()
    isFullscreen.value = await controls.isFullscreen()
  }

  async function minimizeWindow(): Promise<void> {
    await controls.minimize()
  }

  async function toggleMaximizeWindow(): Promise<void> {
    await controls.toggleMaximize()
    await readWindowState().catch(ignoreFailure)
  }

  async function closeWindow(): Promise<void> {
    await controls.close()
  }

  runInBackground(
    controls.getEnvironment().then(({ platform: current, version }) => {
      platform.value = current
      macVersion.value = readMajorVersion(version)
    }),
    ignoreFailure,
  )

  runInBackground(readWindowState(), ignoreFailure)

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
