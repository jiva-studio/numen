import { createApp } from 'vue'
import DemoWindow from '@demo-window'

export function mountDemoWindow(selector: string) {
  const host = document.querySelector(selector)
  if (host) {
    createApp(DemoWindow).mount(host)
  }
}
