import { createApp } from 'vue'
import App from './App.vue'

// The window draws its own menus. The one the webview draws carries a browser's
// idea of what is here — inspect, reload, view source — and refusing it
// everywhere leaves the right button to the application.
document.addEventListener('contextmenu', (event) => event.preventDefault())

// The window's own runtime, which is what carries a file let go of over the
// page back to the application and marks the place it would land. It is served
// beside the page by the window, and a browser reading the page has none.
const runtime = '/wails/runtime.js'
void import(/* @vite-ignore */ runtime).catch(() => {})

createApp(App).mount('#app')
