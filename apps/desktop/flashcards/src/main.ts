import { createApp } from 'vue'
import { interceptExternalLinks } from '@numen/ui'
import App from './app/App.vue'
import './app/app.css'

// The webview's default menu carries browser entries (inspect, reload, view source),
// and none of them belongs in an application. Over selected text the system menu is
// kept, so the selection can be copied.
document.addEventListener('contextmenu', (event) => {
  if (!document.getSelection()?.isCollapsed) return
  event.preventDefault()
})

// A deck may come from another source, and this window has no address bar to show
// where it has ended up. Nothing navigates away from the served pages: an outward
// link is opened where the user opens everything else, through the runtime the window
// serves beside the page.
const runtime = '/wails/runtime.js'
const wails = import(/* @vite-ignore */ runtime).catch(() => undefined)
interceptExternalLinks((href) => {
  void wails.then((it) => it?.Browser.OpenURL(href))
})

createApp(App).mount('#app')
