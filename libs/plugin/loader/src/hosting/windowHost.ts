/**
 * The window the plugins of a desktop window run in: the state of its frame and the window buttons,
 * which the frame carries out. Both windows give their plugins this one host of their frame.
 */
import { computed, watch, type Ref } from 'vue'
import type { WindowHost, WindowState, Caught } from '@numen/plugin-sdk'

/** The part of the frame of a window the host reads and drives. */
export interface WindowControls {
  readonly isMaximized: Readonly<Ref<boolean>>
  readonly isFullscreen: Readonly<Ref<boolean>>
  minimizeWindow(): Promise<void>
  toggleMaximizeWindow(): Promise<void>
  closeWindow(): Promise<void>
}

/** The host of the frame; a button whose call fails passes the failure to `onError`. */
export function createWindowHost(
  frame: WindowControls,
  onError: (error: Caught) => void,
): WindowHost {
  const state = computed<WindowState>(() => ({
    isMaximized: frame.isMaximized.value,
    isFullscreen: frame.isFullscreen.value,
  }))
  const run = (call: () => Promise<void>): void => {
    call().catch(onError)
  }
  return {
    getState: () => state.value,
    watch: (handler) => ({ dispose: watch(state, handler, { immediate: true }) }),
    minimize: () => run(() => frame.minimizeWindow()),
    toggleMaximize: () => run(() => frame.toggleMaximizeWindow()),
    close: () => run(() => frame.closeWindow()),
  }
}
