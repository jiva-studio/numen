import { expect, type Page } from 'playwright/test'

/** The address the index stats item asks for its counts at. */
const STATS_PATH = '/md.numen.tools.index.stats/GetStats'

/** How old a count the core answers can be: `countsMaxAge` in libs/core/kernel/usecase/search/stats.go. */
const COUNTS_MAX_AGE_MS = 1_000

/** The kinds of the files of the test vault. */
const FILE_KINDS = ['PDF', 'EPUB']

/**
 * Waits until the index stats item shows a count the core read after this call, with the files of
 * the vault in it. The core answers every window from a count up to a second old, so the first
 * answer of a window opened right after the cleanup of the test before can still count the vectors
 * of the notes that cleanup removed. On a new server the core reads the files of the vault seconds
 * after its notes. The item asks every 2 seconds only while its popover is open, so the popover
 * stays open until the files are counted and an answer newer than that comes.
 */
export const waitForCurrentVectors = async (page: Page): Promise<void> => {
  const label = page.locator('.index-stats__trigger')
  await label.click()
  const details = page.getByRole('dialog', { name: 'Index stats' })
  for (const kind of FILE_KINDS) {
    await expect(details.locator('dt').getByText(kind, { exact: true })).toBeVisible({
      timeout: 45_000,
    })
  }
  const after = Date.now() + COUNTS_MAX_AGE_MS
  const statsRequest = await page.waitForRequest(
    (request) => request.url().endsWith(STATS_PATH) && Date.now() >= after,
    { timeout: 10_000 },
  )
  const statsResponse = await statsRequest.response()
  await label.click()
  await expect(label).toHaveAttribute('aria-expanded', 'false')
  const { vectors } = ((await statsResponse?.json()) ?? {}) as { vectors?: number }
  await expect
    .poll(async () => Number(((await label.textContent()) ?? '').replace(/\D/g, '')))
    .toBe(vectors ?? 0)
}
