/** A plugin card's filter values, read from its data attributes. */
type CardFilter = { search: string; category: string; apps: string[]; isPaid: boolean; isOfficial: boolean }

/** The values of the filter form. */
type FormFilter = { query: string; category: string; app: string; price: string; isOfficialOnly: boolean }

function readCard(card: HTMLElement): CardFilter {
  return {
    search: card.dataset.search ?? '',
    category: card.dataset.category ?? '',
    apps: (card.dataset.apps ?? '').split(' ').filter(Boolean),
    isPaid: card.dataset.paid === 'true',
    isOfficial: card.dataset.official === 'true',
  }
}

function readForm(form: HTMLFormElement): FormFilter {
  const data = new FormData(form)
  const read = (name: string): string => {
    const value = data.get(name)
    return typeof value === 'string' ? value : ''
  }
  return {
    query: read('query').trim().toLowerCase(),
    category: read('category'),
    app: read('app'),
    price: read('price'),
    isOfficialOnly: data.get('official') !== null,
  }
}

/** Whether a card passes the filter: every search word, and each chosen value. */
export function isCardShown(card: CardFilter, filter: FormFilter): boolean {
  const words = filter.query.split(/\s+/).filter(Boolean)
  if (!words.every((word) => card.search.includes(word))) return false
  if (filter.category !== '' && card.category !== filter.category) return false
  if (filter.app !== '' && !card.apps.includes(filter.app)) return false
  if (filter.price === 'free' && card.isPaid) return false
  if (filter.price === 'paid' && !card.isPaid) return false
  return !filter.isOfficialOnly || card.isOfficial
}

function applyFilter(form: HTMLFormElement, cards: HTMLElement[], empty: HTMLElement | null): void {
  const filter = readForm(form)
  let shown = 0
  for (const card of cards) {
    const isShown = isCardShown(readCard(card), filter)
    card.hidden = !isShown
    if (isShown) shown += 1
  }
  if (empty) empty.hidden = shown > 0
}

/** Wires the search and the filters of the plugin catalog. */
export function initPluginCatalog(): void {
  const form = document.querySelector<HTMLFormElement>('[data-plugin-filters]')
  if (!form) return
  const cards = [...document.querySelectorAll<HTMLElement>('[data-plugin-card]')]
  const empty = document.querySelector<HTMLElement>('[data-plugin-empty]')
  const onChange = (): void => applyFilter(form, cards, empty)
  form.addEventListener('input', onChange)
  form.addEventListener('change', onChange)
  form.addEventListener('submit', (event) => event.preventDefault())
  onChange()
}
