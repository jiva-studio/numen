import { createApp } from 'vue'
import { createKeepaliveTransport, createOriginTransport } from '@numen/core-client'
import { reportFailure, runInBackground } from '@numen/ui'
import { createFrameRuntimeLoader, openExternalUrl } from '@numen/plugin-loader'
import { interceptExternalLinks } from './features/external-links'
import { createRoot } from './app/createRoot'

// The window draws its own menus. The one the webview draws carries a browser's
// idea of what is here — inspect, reload, view source — and refusing it
// everywhere leaves the right button to the application.
document.addEventListener('contextmenu', (event) => event.preventDefault())

const loadFrameRuntime = createFrameRuntimeLoader()

// A note can contain any link, and this window has no address bar to show the
// current location. Links stay inside the served pages, and an external
// address opens in the system browser.
interceptExternalLinks((href) => {
  runInBackground(openExternalUrl(href, loadFrameRuntime), reportFailure)
})

createApp(
  createRoot({ transport: createOriginTransport(), keepalive: createKeepaliveTransport() }),
).mount('#app')
