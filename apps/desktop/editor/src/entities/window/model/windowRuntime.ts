/**
 * Focus and close requests for the desktop window, sent through the Wails runtime.
 */
import { watch, type Ref } from 'vue'
import type { FrameRuntimeLoader } from '../api/frameRuntime'

export async function focusWindow(load: FrameRuntimeLoader): Promise<void> {
  await (await load())?.Window?.Focus?.()
}

/** Closes the window through the same path as its close button. */
export async function closeWindow(load: FrameRuntimeLoader): Promise<void> {
  await (await load())?.Window?.Close?.()
}

export interface EmptyWindowCloseDeps {
  readonly isLayoutReady: Readonly<Ref<boolean>>
  readonly countTabs: () => number
  readonly closeWindow: () => Promise<void>
  readonly onError: (error: unknown) => void
}

/** Closes the window when the layout is ready and the tab count drops to zero. */
export function closeWindowWhenEmpty(deps: EmptyWindowCloseDeps): void {
  watch(
    () => deps.isLayoutReady.value && deps.countTabs() === 0,
    (isEmpty) => {
      if (isEmpty) deps.closeWindow().catch(deps.onError)
    },
  )
}
