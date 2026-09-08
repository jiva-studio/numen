/** What the window keeps a url tab by, and what draws it. */
import type { Medium } from '../shared/media/kind'
import { URL } from '../shared/tabs/workspace'
import UrlTab from './UrlTab.vue'

/** The urls of the vault, opened at what is at the address. */
export const URLS: Medium = {
  tab: URL,
  source: 'url',
  draws: UrlTab,
  hands: (puts, opens) => puts.points(opens),
}
