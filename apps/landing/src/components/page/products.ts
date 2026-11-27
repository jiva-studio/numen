/** The products the site has a page for, and those that are announced. */
export type ProductId = 'numen' | 'flashcards' | 'sync'

type Product = {
  id: ProductId
  name: string
  sub: string
  href: string
  icon: string
  key: string
  text: string
}

type AnnouncedProduct = {
  id: string
  name: string
  glyph: string
  key: string
  text: string
}

export const PRODUCTS: readonly Product[] = [
  {
    id: 'numen',
    name: 'Numen',
    sub: 'The Second Brain',
    href: '/',
    icon: '/favicon.svg',
    key: 'product_numen',
    text: 'Write notes and link them into a map of what you know.',
  },
  {
    id: 'flashcards',
    name: 'Numen Flashcards',
    sub: 'Flashcards',
    href: '/flashcards',
    icon: '/flashcards.svg',
    key: 'product_flashcards',
    text: 'Remember what is in your notes, a few minutes a day.',
  },
  {
    id: 'sync',
    name: 'Numen Sync',
    sub: 'Sync',
    href: '/sync',
    icon: '/sync.svg',
    key: 'product_sync',
    text: 'Your vaults on every device, end-to-end encrypted.',
  },
]

export const ANNOUNCED_PRODUCTS: readonly AnnouncedProduct[] = [
  {
    id: 'mobile',
    name: 'Numen Mobile',
    glyph: '<rect x="7" y="3" width="10" height="18" rx="2.5"/><path d="M11 18h2"/>',
    key: 'product_mobile',
    text: 'All your notes, always at hand, on your phone.',
  },
  {
    id: 'books',
    name: 'Numen Books',
    glyph:
      '<path d="M4 5.5A1.5 1.5 0 0 1 5.5 4H11v16H5.5A1.5 1.5 0 0 1 4 18.5Z"/><path d="M20 5.5A1.5 1.5 0 0 0 18.5 4H13v16h5.5a1.5 1.5 0 0 0 1.5-1.5Z"/>',
    key: 'product_books',
    text: 'Read books and keep them in your second brain.',
  },
]

/** The plugins the products panel shows beside the products. */
export const MENU_PLUGINS = [
  {
    id: 'md.numen.views.plex.graph',
    icon: 'waypoints',
    name: 'Plex Graph',
    key: 'menu_plex',
    text: 'A map of connected notes and topics.',
  },
  {
    id: 'md.numen.engines.transcription',
    icon: 'audio-lines',
    name: 'Transcription',
    key: 'menu_transcription',
    text: 'Speech to text, offline.',
  },
  {
    id: 'md.numen.agents.numen',
    icon: 'bot',
    name: 'Numen Agent',
    key: 'menu_agent',
    text: 'An AI assistant, also on local models.',
  },
]

/** The product a page is about. */
export function getProduct(id: ProductId): Product {
  const product = PRODUCTS.find((one) => one.id === id)
  if (!product) throw new Error(`unknown product ${id}`)
  return product
}
