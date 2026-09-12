/** What the window keeps a url tab by, and what draws it. */
import type { Medium } from '@/entities/media/kind'
import { URL } from '@/entities/tab/workspace'
import UrlTab from './ui/UrlTab.vue'

/** The urls of the vault, opened at what is at the address. */
export const URLS: Medium<typeof URL> = {
  tab: URL,
  source: 'url',
  draws: UrlTab,
  hands: (puts, opens) => {
    // A url is reached both ways: by what the vault says stands at a path,
    // and by having just been made here.
    puts.reads({ kind: URLS.source }, opens)
    puts.holds('url', (path) => opens(path, []))
  },
}
