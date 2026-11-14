import { createApp } from 'vue'
import { interceptExternalLinks, reportFailure, runInBackground } from '@numen/ui'
import { openExternalUrl } from '@numen/workbench'
import { createRoot } from './app/createRoot'
import { loadFrameRuntime } from './app/runtime'
import './app/app.css'

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

createApp(createRoot()).mount('#app')
