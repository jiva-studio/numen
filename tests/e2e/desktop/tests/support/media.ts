/**
 * What the tests of decks, recordings, links, the agent and the updater need beyond `vault.ts`:
 * files copied and removed on disk, the folders plugins keep in the vault, deck files, and the
 * stand-ins' address.
 */
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import type { APIRequestContext } from 'playwright/test'
import { getVaultPath, readOwner } from './vault.js'

/** Whether a file of the vault is on disk. */
export const hasVaultFile = (file: string): boolean =>
  fs.existsSync(path.join(getVaultPath(), file))

/** The letters a card id of a deck is written in (the idPattern of the flashcards manifest). */
const CARD_ID_LETTERS = '0123456789abcdefghjkmnpqrstvwxyz'

/** A card id no fixture of the vault carries. */
export const createCardId = (): string =>
  Array.from(
    { length: 10 },
    () => CARD_ID_LETTERS[Math.floor(Math.random() * CARD_ID_LETTERS.length)],
  ).join('')

/** One card of a deck file: its id, the card model that cuts it and its values by field. */
export interface DeckCard {
  readonly id: string
  readonly cardModel: string
  readonly values: Readonly<Record<string, string>>
}

/** The text of a deck file holding the cards under its title, before any section. */
export const createDeckText = (title: string, cards: readonly DeckCard[]): string =>
  [
    `---\ntype: deck\ntitle: ${title}\n---`,
    ...cards.map((card) =>
      [
        `## ${Object.values(card.values)[0] ?? ''} ^${card.id}`,
        `[[${card.cardModel}]]`,
        ...Object.entries(card.values).map(([field, value]) => `### ${field}\n\n${value}`),
      ].join('\n\n'),
    ),
  ].join('\n\n') + '\n'

/**
 * The origin of the stand-ins `scripts/stand-ins.mjs` starts, read from the release feed address
 * the run's settings give the updater plugin.
 */
export const getStandInsOrigin = async (request: APIRequestContext): Promise<string> => {
  const endpoint = String((await readOwner(request, 'md.numen.tools.updater'))['endpoint'] ?? '')
  if (!endpoint) throw new Error('the settings give the updater no release feed address')
  return new URL(endpoint).origin
}

const getPageTextsFile = (): string =>
  path.join(os.tmpdir(), `numen-e2e-page-texts-${process.env.NUMEN_TEST_PORT ?? '3456'}.json`)

const readPageTexts = (): Record<string, string> => {
  try {
    return JSON.parse(fs.readFileSync(getPageTextsFile(), 'utf8')) as Record<string, string>
  } catch {
    return {}
  }
}

/** Makes the page stand-in answer the address with the text in the first paragraph of its page. */
export const writePageText = (address: string, text: string): void => {
  fs.writeFileSync(getPageTextsFile(), JSON.stringify({ ...readPageTexts(), [address]: text }))
}

/** Makes the page stand-in answer the address with its own page again. */
export const removePageText = (address: string): void => {
  const { [address]: _removed, ...remaining } = readPageTexts()
  fs.writeFileSync(getPageTextsFile(), JSON.stringify(remaining))
}
