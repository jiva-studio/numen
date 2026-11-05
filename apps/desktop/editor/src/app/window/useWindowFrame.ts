/**
 * The frame of the desktop window, drawn by the page: minimise, maximise, close, and the platform
 * the buttons follow. The calls go through the Wails runtime served beside the page; a page served
 * without it (stories, tests) gets a frame that does nothing.
 */
import { computed, getCurrentScope, onScopeDispose, ref, shallowRef, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { ControlsStyle } from '@numen/shell'

/** The values of the setting "Window buttons". */
export type WindowButtons = Extract<ControlsStyle, 'windows' | 'mac'>

/** The path the window serves its Wails runtime module at. */
const RUNTIME_PATH = '/wails/runtime.js'

/** The part of the Wails runtime module the frame uses. */
export interface FrameRuntime {
  readonly Window?: {
    readonly Minimise?: () => Promise<void>
    readonly ToggleMaximise?: () => Promise<void>
    readonly Close?: () => Promise<void>
    readonly IsMaximised?: () => Promise<boolean>
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
  /** How the title bar draws the window buttons on this platform. */
  readonly controls: Readonly<Ref<ControlsStyle>>
  readonly isMaximised: Readonly<Ref<boolean>>
  minimise(): Promise<void>
  toggleMaximise(): Promise<void>
  close(): Promise<void>
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

/**
 * Windows draws the buttons at the right to keep Snap Layouts, and macOS keeps the system buttons.
 * Linux draws them where the setting "Window buttons" says.
 */
function getControls(system: FrameRuntime['System'], buttons: WindowButtons): ControlsStyle {
  if (system?.IsMac?.()) return 'native-mac'
  return system?.IsLinux?.() ? buttons : 'windows'
}

/** A read of the window state that fails leaves the state as it was. */
const ignoreFailure = (): undefined => undefined

export function useWindowFrame(
  load: FrameRuntimeLoader = loadRuntime,
  windowButtons: Readonly<Ref<WindowButtons>> = ref<WindowButtons>('windows'),
): WindowFrame {
  const system = shallowRef<FrameRuntime['System']>()
  const isMaximised = ref(false)

  async function readMaximised(): Promise<void> {
    isMaximised.value = (await (await load())?.Window?.IsMaximised?.()) ?? false
  }

  async function minimise(): Promise<void> {
    await (await load())?.Window?.Minimise?.()
  }

  async function toggleMaximise(): Promise<void> {
    await (await load())?.Window?.ToggleMaximise?.()
    await readMaximised().catch(ignoreFailure)
  }

  async function close(): Promise<void> {
    await (await load())?.Window?.Close?.()
  }

  runInBackground(
    load().then(async (loaded) => {
      system.value = loaded?.System
      await readMaximised()
    }),
    ignoreFailure,
  )

  // A resize is how a window gets maximised or restored by the system, not only by the button.
  const onResize = () => runInBackground(readMaximised(), ignoreFailure)
  globalThis.addEventListener('resize', onResize)
  if (getCurrentScope()) onScopeDispose(() => globalThis.removeEventListener('resize', onResize))

  return {
    controls: computed(() => getControls(system.value, windowButtons.value)),
    isMaximised,
    minimise,
    toggleMaximise,
    close,
  }
}
