import { defineComponent, h } from 'vue'
import type { Transport } from '@/shared/api/clients'
import App from './App.vue'
import { createWindow, type WindowDeps } from './window/createWindow'

/**
 * The root of the page: it makes the window inside its own setup, because the window starts and
 * stops with the component that holds it, and hands it to the component that draws it. The window
 * talks to the core over `transport`.
 */
export const createRoot = (transport: Transport, deps?: WindowDeps) =>
  defineComponent({
    setup: () => ({ parts: createWindow(transport, deps) }),
    render() {
      return h(App, { parts: this.parts })
    },
  })
