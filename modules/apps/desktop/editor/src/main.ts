import { createApp } from 'vue'
import { interceptExternalLinks } from '@numen/ui'
import App from './app/shell/App.vue'

// The window draws its own menus. The one the webview draws carries a browser's
// idea of what is here — inspect, reload, view source — and refusing it
// everywhere leaves the right button to the application.
document.addEventListener('contextmenu', (event) => event.preventDefault())

// The window's own runtime, which is what carries a file release over the
// page back to the application and marks the place it would land. It is served
// beside the page by the window, and a browser reading the page has none.
const runtime = '/wails/runtime.js'
const wails = import(/* @vite-ignore */ runtime).catch(() => undefined)

// A note may have been written by anybody, and this window has no address bar
// to say where it has ended up. Nothing takes it off the pages it serves: an
// address that leads outward is opened where the person opens everything else.
interceptExternalLinks((href) => {
  void wails.then((it) => {
    if (it?.Browser?.OpenURL) {
      it.Browser.OpenURL(href)
    } else {
      window.open(href, '_blank')
    }
  })
})

createApp(App).mount('#app')
