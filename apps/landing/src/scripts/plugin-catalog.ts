/** A catalog row's filter values, read from its data attributes. */
type EntryFilter = { search: string; category: string }

/** The search text and the chosen category; an empty category chooses every one. */
export type CatalogFilter = { query: string; category: string }

/** Whether a row passes the filter: every search word, and the chosen category. */
export function isEntryShown(entry: EntryFilter, filter: CatalogFilter): boolean {
  const words = filter.query.trim().toLowerCase().split(/\s+/).filter(Boolean)
  if (!words.every((word) => entry.search.includes(word))) return false
  return filter.category === '' || entry.category === filter.category
}

function showMatchingEntries(filter: CatalogFilter): void {
  let shown = 0
  for (const row of document.querySelectorAll<HTMLElement>('[data-plugin-entry]')) {
    const entry = { search: row.dataset.search ?? '', category: row.dataset.category ?? '' }
    const isShown = isEntryShown(entry, filter)
    row.hidden = !isShown
    if (isShown) shown += 1
  }
  const empty = document.querySelector<HTMLElement>('[data-plugin-empty]')
  if (empty) empty.hidden = shown > 0
  const featured = document.querySelector<HTMLElement>('[data-plugin-featured]')
  if (featured) featured.hidden = filter.query.trim() !== ''
}

/** Wires the search field and the category links of the catalog to the rows of all plugins. */
export function initPluginCatalog(): void {
  const filter: CatalogFilter = { query: '', category: '' }
  const search = document.querySelector<HTMLInputElement>('[data-plugin-search]')
  const categories = [...document.querySelectorAll<HTMLButtonElement>('[data-plugin-category]')]
  search?.addEventListener('input', () => {
    filter.query = search.value
    showMatchingEntries(filter)
  })
  search?.addEventListener('keydown', (event) => {
    if (event.key !== 'Escape') return
    search.value = ''
    filter.query = ''
    showMatchingEntries(filter)
  })
  for (const button of categories) {
    button.addEventListener('click', () => {
      filter.category = button.dataset.pluginCategory ?? ''
      for (const one of categories) one.setAttribute('aria-pressed', String(one === button))
      showMatchingEntries(filter)
    })
  }
  if (search?.value) {
    filter.query = search.value
    showMatchingEntries(filter)
  }
}
