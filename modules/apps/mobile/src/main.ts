/**
 * The phone's application.
 *
 * A link leading out of it is the platform's: the WebView hands an address off
 * the page's own host to whatever opens one on the phone, and loads none of it
 * itself. Holding the window in the page, as the desktop windows do, would take
 * that away and put nothing in its place; `server.allowNavigation` widens it.
 */
import { createApp } from 'vue'
import { IonicVue } from '@ionic/vue'

import '@ionic/vue/css/core.css'
import '@ionic/vue/css/normalize.css'
import '@ionic/vue/css/structure.css'
import '@ionic/vue/css/typography.css'
import '@numen/ui/styles.css'

import App from './App.vue'

createApp(App).use(IonicVue).mount('#app')
