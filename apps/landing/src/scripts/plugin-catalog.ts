/** A plugin card's filter values, read from its data attributes. */
type CardFilter = { search: string; category: string; apps: string[]; isPaid: boolean; isOfficial: boolean }

/** The values of the filter bar; an empty string chooses every value. */
export type CatalogFilter = {
  query: string
  category: string
  app: string
  price: string
  isOfficialOnly: boolean
}

function readCard(card: HTMLElement): CardFilter {
  return {
    search: card.dataset.search ?? '',
    category: card.dataset.category ?? '',
    apps: (card.dataset.apps ?? '').split(' ').filter(Boolean),
    isPaid: card.dataset.paid === 'true',
    isOfficial: card.dataset.official === 'true',
  }
}

/** Whether a card passes the filter: every search word, and each chosen value. */
export function isCardShown(card: CardFilter, filter: CatalogFilter): boolean {
  const words = filter.query.trim().toLowerCase().split(/\s+/).filter(Boolean)
  if (!words.every((word) => card.search.includes(word))) return false
  if (filter.category !== '' && card.category !== filter.category) return false
  if (filter.app !== '' && !card.apps.includes(filter.app)) return false
  if (filter.price === 'free' && card.isPaid) return false
  if (filter.price === 'paid' && !card.isPaid) return false
  return !filter.isOfficialOnly || card.isOfficial
}

/** Hides the catalog cards the filter does not pass, and shows the empty note when none is left. */
export function showMatchingCards(filter: CatalogFilter): void {
  let shown = 0
  for (const card of document.querySelectorAll<HTMLElement>('[data-plugin-card]')) {
    const isShown = isCardShown(readCard(card), filter)
    card.hidden = !isShown
    if (isShown) shown += 1
  }
  const empty = document.querySelector<HTMLElement>('[data-plugin-empty]')
  if (empty) empty.hidden = shown > 0
}

/** Calls `onChange` with the page language now and whenever it changes; returns the stop function. */
export function watchPageLanguage(onChange: (lang: string) => void): () => void {
  const root = document.documentElement
  const observer = new MutationObserver(() => onChange(root.lang))
  observer.observe(root, { attributes: true, attributeFilter: ['lang'] })
  onChange(root.lang)
  return () => observer.disconnect()
}
