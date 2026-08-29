import { createApp } from 'vue'
import App from './App.vue'
import './app.css'

// The menu the webview draws carries a browser's idea of what is here —
// inspect, reload, view source — and none of it belongs in an application. Over
// text a person has selected the machine's own menu stands, because copying
// what a card says is theirs to do.
document.addEventListener('contextmenu', (event) => {
  if (!document.getSelection()?.isCollapsed) return
  event.preventDefault()
})

createApp(App).mount('#app')
