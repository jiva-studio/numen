import { createApp } from 'vue'
import { createRoot } from './app/createRoot'
import './app/app.css'

// The webview's default menu carries browser entries (inspect, reload, view source),
// and none of them belongs in an application. Over selected text the system menu is
// kept, so the selection can be copied.
document.addEventListener('contextmenu', (event) => {
  if (!document.getSelection()?.isCollapsed) return
  event.preventDefault()
})

createApp(createRoot()).mount('#app')
