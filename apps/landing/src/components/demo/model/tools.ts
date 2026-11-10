import { onBeforeUnmount, reactive } from 'vue'
import { BOLTZMANN_PATH, PDF_DOCS, SHANNON_PATH } from '../lib/documents'
import { getPdfDoc } from '../lib/pdfPage'
import { createAgent } from './agent'
import { createDeck } from './deck'
import { createPlayer } from './player'

/** The tools of the demo window that hold their own state: the agent, the deck, the player and the pages of the PDF readers. */
export function createDemoTools(getFocusTitle: () => string) {
  const agent = createAgent(getFocusTitle)
  const deck = createDeck()
  const player = createPlayer()
  const pages = reactive<Record<string, number>>({
    [BOLTZMANN_PATH]: PDF_DOCS[BOLTZMANN_PATH].firstPage,
    [SHANNON_PATH]: PDF_DOCS[SHANNON_PATH].firstPage,
  })

  const goToPage = (path: string, page: number) => {
    pages[path] = Math.min(Math.max(page, 0), getPdfDoc(path).leaves - 1)
  }

  onBeforeUnmount(() => player.pause())

  return { agent, deck, player, pdf: { pages, goToPage } }
}
