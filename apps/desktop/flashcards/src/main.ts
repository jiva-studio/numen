import { createApp } from 'vue'
import App from './app/App.vue'
import './app/app.css'

// The webview's default menu carries browser entries (inspect, reload, view source),
// and none of them belongs in an application. Over selected text the system menu is
// kept, so the selection can be copied.
document.addEventListener('contextmenu', (event) => {
  if (!document.getSelection()?.isCollapsed) return
  event.preventDefault()
})

createApp(App).mount('#app')
