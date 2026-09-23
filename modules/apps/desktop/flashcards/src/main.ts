import { createApp } from 'vue'
import { holdWindow } from '@numen/ui'
import App from './app/App.vue'
import './app/app.css'

// The menu the webview draws carries a browser's idea of what is here —
// inspect, reload, view source — and none of it belongs in an application. Over
// text a person has selected the machine's own menu stands, because copying
// what a card says is theirs to do.
document.addEventListener('contextmenu', (event) => {
  if (!document.getSelection()?.isCollapsed) return
  event.preventDefault()
})

// A deck may have come from another person, and this window has no address bar
// to say where it has ended up. Nothing takes it off the pages it serves: an
// address that leads outward is opened where the person opens everything else,
// through the runtime the window serves beside the page.
const runtime = '/wails/runtime.js'
const wails = import(/* @vite-ignore */ runtime).catch(() => undefined)
holdWindow((href) => {
  void wails.then((it) => it?.Browser.OpenURL(href))
})

createApp(App).mount('#app')
