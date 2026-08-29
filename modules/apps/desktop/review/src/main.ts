import { createApp } from 'vue'
import App from './App.vue'
import './app.css'

// The window draws its own. The menu the webview draws carries a browser's idea
// of what is here — inspect, reload, view source — and refusing it everywhere
// leaves the right button to the application.
document.addEventListener('contextmenu', (event) => event.preventDefault())

createApp(App).mount('#app')
