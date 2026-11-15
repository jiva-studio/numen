import { defineComponent, h } from 'vue'
import type { WindowTransports } from '@/shared/api/clients'
import App from './App.vue'
import { createWindow } from './window/createWindow'

/**
 * The root of the page: it makes the window inside its own setup, because the window starts and
 * stops with the component that holds it, and hands it to the component that draws it. The window
 * talks to the core over `transports`.
 */
export const createRoot = (transports: WindowTransports, makeWindow = createWindow) =>
  defineComponent({
    setup: () => ({ parts: makeWindow(transports) }),
    render() {
      return h(App, { parts: this.parts })
    },
  })
