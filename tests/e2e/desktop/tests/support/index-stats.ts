import { expect, type Page } from 'playwright/test'

/** The address the index stats item asks for its counts at. */
const STATS_PATH = '/md.numen.tools.index.stats/GetStats'

/** How old a count the core answers can be: `countsMaxAge` in libs/core/kernel/adapter/plugin/hostsearch/stats.go. */
const COUNTS_MAX_AGE_MS = 1_000

/**
 * Waits until the index stats item shows a count the core read after this call. The core answers
 * every window from a count up to a second old, so the first answer of a window opened right after
 * the cleanup of the test before can still count the vectors of the notes that cleanup removed.
 */
export const waitForCurrentVectors = async (page: Page): Promise<void> => {
  const after = Date.now() + COUNTS_MAX_AGE_MS
  const statsRequest = await page.waitForRequest(
    (request) => request.url().endsWith(STATS_PATH) && Date.now() >= after,
    { timeout: 10_000 },
  )
  const statsResponse = await statsRequest.response()
  const { vectors } = ((await statsResponse?.json()) ?? {}) as { vectors?: number }
  const label = page.locator('.index-stats__trigger')
  await expect
    .poll(async () => Number(((await label.textContent()) ?? '').replace(/\D/g, '')))
    .toBe(vectors ?? 0)
}

/** The part of an answer of the index stats item a spec reads. */
interface IndexStats {
  readonly waiting: number
  readonly kinds: readonly { readonly kind: string; readonly chunks: number }[]
}

/** The counts of the notes in an answer of the index stats item. */
export interface NoteCounts {
  readonly chunks: number
  readonly isEmbedded: boolean
}

/**
 * Waits for the next answer of the index stats item and returns the counts of its notes. A spec that
 * compares the chunks of notes reads them apart from the file texts, whose texts the plugins can
 * write after the window opens.
 */
export const readNoteCounts = async (page: Page): Promise<NoteCounts> => {
  const response = await page.waitForResponse((one) => one.url().endsWith(STATS_PATH), {
    timeout: 10_000,
  })
  const stats = (await response.json()) as IndexStats
  const notes = stats.kinds.find((one) => one.kind === 'notes')
  return { chunks: notes?.chunks ?? 0, isEmbedded: stats.waiting === 0 }
}
