import { expect, type Page } from 'playwright/test'

/** The address the index stats item asks for its counts at. */
const STATS_PATH = '/md.numen.tools.index.stats/GetStats'

/** How old a count the core answers can be: `countsMaxAge` in libs/core/kernel/usecase/search/stats.go. */
const COUNTS_MAX_AGE_MS = 1_000

/**
 * Waits until the index stats item shows a count the core read after this call. The core answers
 * every window from a count up to a second old, so the first answer of a window opened right after
 * the cleanup of the test before can still count the vectors of the notes that cleanup removed. The
 * item asks every 2 seconds only while its popover is open, so the popover stays open until an answer
 * newer than that comes.
 */
export const waitForCurrentVectors = async (page: Page): Promise<void> => {
  const after = Date.now() + COUNTS_MAX_AGE_MS
  const label = page.locator('.index-stats__trigger')
  const statsRequest = page.waitForRequest(
    (request) => request.url().endsWith(STATS_PATH) && Date.now() >= after,
    { timeout: 10_000 },
  )
  await label.click()
  const statsResponse = await (await statsRequest).response()
  await label.click()
  await expect(label).toHaveAttribute('aria-expanded', 'false')
  const { vectors } = ((await statsResponse?.json()) ?? {}) as { vectors?: number }
  await expect
    .poll(async () => Number(((await label.textContent()) ?? '').replace(/\D/g, '')))
    .toBe(vectors ?? 0)
}
