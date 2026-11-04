/**
 * Focus and close requests for the desktop window, sent through the Wails runtime.
 */
import { watch, type Ref } from 'vue'

/** The path the window serves its Wails runtime module at. */
const RUNTIME_PATH = '/wails/runtime.js'

/** The Wails runtime API for the current window. */
export interface WindowRuntime {
  readonly Window?: {
    readonly Focus?: () => Promise<void>
    readonly Close?: () => Promise<void>
  }
}

/** Loads the runtime module; resolves to undefined where the page is not served by Wails. */
export type WindowRuntimeLoader = () => Promise<WindowRuntime | undefined>

let runtime: Promise<WindowRuntime | undefined> | undefined

// Stryker disable next-line all: dynamic import of /wails/runtime.js is only resolvable in the Wails webview
const loadRuntime: WindowRuntimeLoader = () => {
  runtime ??= import(/* @vite-ignore */ RUNTIME_PATH).then(
    (loaded: WindowRuntime) => loaded,
    () => undefined,
  )
  return runtime
}

/** Focuses the window. */
export async function focusWindow(load: WindowRuntimeLoader = loadRuntime): Promise<void> {
  await (await load())?.Window?.Focus?.()
}

/** Closes the window through the same path as its close button. */
export async function closeWindow(load: WindowRuntimeLoader = loadRuntime): Promise<void> {
  await (await load())?.Window?.Close?.()
}

export interface EmptyWindowCloseDeps {
  /** Whether the initial tabs of the window are open. */
  readonly isLayoutReady: Readonly<Ref<boolean>>
  readonly tabCount: () => number
  readonly closeWindow: () => Promise<void>
  readonly onError: (error: unknown) => void
}

/** Closes the window when the layout is ready and the tab count drops to zero. */
export function closeWindowWhenEmpty(deps: EmptyWindowCloseDeps): void {
  watch(
    () => deps.isLayoutReady.value && deps.tabCount() === 0,
    (isEmpty) => {
      if (isEmpty) deps.closeWindow().catch(deps.onError)
    },
  )
}
