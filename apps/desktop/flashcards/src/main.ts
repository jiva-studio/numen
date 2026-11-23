import { createApp } from 'vue'
import { createOriginTransport } from '@numen/core-client'
import { reportFailure, runInBackground } from '@numen/ui'
import { createFrameRuntimeLoader, openExternalUrl } from '@numen/plugin-loader'
import { interceptExternalLinks } from './features/external-links'
import { createRoot } from './app/createRoot'
import './app/app.css'

const loadFrameRuntime = createFrameRuntimeLoader()

// The webview's default menu carries browser entries (inspect, reload, view source),
// and none of them belongs in an application. Over selected text the system menu is
// kept, so the selection can be copied.
document.addEventListener('contextmenu', (event) => {
  if (!document.getSelection()?.isCollapsed) return
  event.preventDefault()
})

// The window has no address bar: an external address opens in the system browser.
interceptExternalLinks((href) => {
  runInBackground(openExternalUrl(href, loadFrameRuntime), reportFailure)
})

createApp(createRoot(createOriginTransport())).mount('#app')
