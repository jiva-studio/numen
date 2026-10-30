import { createApp } from 'vue'
import { interceptExternalLinks, runInBackground } from '@numen/ui'
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

// A note can contain any link, and this window has no address bar to show the
// current location. Links stay inside the served pages, and an external
// address opens in the system browser.
interceptExternalLinks((href) => {
  const openInBrowser = () => window.open(href, '_blank')
  runInBackground(
    wails.then((it) => {
      if (it?.Browser?.OpenURL) {
        it.Browser.OpenURL(href)
      } else {
        openInBrowser()
      }
    }),
    openInBrowser,
  )
})

createApp(App).mount('#app')
