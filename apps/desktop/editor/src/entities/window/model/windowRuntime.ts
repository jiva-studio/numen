/**
 * Focus and close requests for the desktop window, sent through the Wails runtime.
 */
import { watch, type Ref } from 'vue'
import { loadWailsRuntime } from '@/shared/api/wailsRuntime'
import type { WailsRuntimeLoader } from '@/shared/lib/runtime/wailsRuntime'
import type { Caught } from '@/shared/api/errors'

/** Loads the runtime module; resolves to undefined where the page is not served by Wails. */
type WindowRuntimeLoader = WailsRuntimeLoader

/** Focuses the window. */
export async function focusWindow(load: WindowRuntimeLoader = loadWailsRuntime): Promise<void> {
  await (await load())?.Window?.Focus?.()
}

/** Closes the window through the same path as its close button. */
export async function closeWindow(load: WindowRuntimeLoader = loadWailsRuntime): Promise<void> {
  await (await load())?.Window?.Close?.()
}

export interface EmptyWindowCloseDeps {
  readonly isLayoutReady: Readonly<Ref<boolean>>
  readonly tabCount: () => number
  readonly closeWindow: () => Promise<void>
  readonly onError: (error: Caught) => void
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
