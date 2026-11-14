import { createApp } from 'vue'
import { interceptExternalLinks, reportFailure, runInBackground } from '@numen/ui'
import { openExternalUrl } from '@numen/workbench'
import { createRoot } from './app/createRoot'
import { loadFrameRuntime } from './shared/api/runtime'

// The window draws its own menus. The one the webview draws carries a browser's
// idea of what is here — inspect, reload, view source — and refusing it
// everywhere leaves the right button to the application.
document.addEventListener('contextmenu', (event) => event.preventDefault())

// A note can contain any link, and this window has no address bar to show the
// current location. Links stay inside the served pages, and an external
// address opens in the system browser.
interceptExternalLinks((href) => {
  runInBackground(openExternalUrl(href, loadFrameRuntime), reportFailure)
})

createApp(createRoot()).mount('#app')
