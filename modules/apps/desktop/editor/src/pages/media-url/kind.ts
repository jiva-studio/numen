/** What the window keeps a url tab by, and what draws it. */
import type { Medium } from '@/entities/media'
import { URL } from '@/entities/tab'
import UrlTab from './ui/UrlTab.vue'

/** The urls of the vault, opened at what is at the address. */
export const URLS: Medium<typeof URL> = {
  tab: URL,
  source: 'url',
  draws: UrlTab,
  hands: (tabOpeners, opens) => {
    // A url is reached both ways: by what the vault says stands at a path,
    // and by having just been made here.
    tabOpeners.registerReader({ kind: URLS.source }, opens)
    tabOpeners.registerEditor('url', (path) => opens(path, []))
  },
}
