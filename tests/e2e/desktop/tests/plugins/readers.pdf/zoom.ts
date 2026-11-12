/** What the PDF zoom specs share: opening the pages, and holding the requests for page images. */
import type { Locator, Page, Request } from 'playwright/test'
import { expect } from '../../support/test.js'
import { openWorkspace, openVaultFile } from '../../support/workspace.js'
import { readSettledWidth } from '../../support/layout.js'

/** The route of the page images. */
const PAGES = '**/plugin/md.numen.readers.pdf/pages/**'

export const getPageIndex = (url: string): number => Number(/pages\/(\d+)/.exec(url)?.[1])
export const getWidePageWidth = (url: string): number =>
  Number(new URL(url, 'https://x').searchParams.get('wide'))

/** Page requests are held until the test releases them, so a test sees what the window shows meanwhile. */
export const holdPageRequests = async (page: Page) => {
  const requested: Request[] = []
  const failed: string[] = []
  let release!: () => void
  const gate = new Promise<void>((resolve) => {
    release = resolve
  })
  page.on('requestfailed', (request) => failed.push(request.url()))
  await page.route(PAGES, async (route) => {
    requested.push(route.request())
    await gate
    await route.continue().catch(() => undefined)
  })
  return { requested, failed, release }
}

export const readNaturalWidth = (picture: Locator) =>
  picture.evaluate((one) => (one as HTMLImageElement).naturalWidth)

export const openPages = async (page: Page) => {
  await openWorkspace(page)
  await openVaultFile(page, 'assets', 'pages.pdf')
  const first = page.locator('.reader__page[data-page="0"]')
  await expect(first.locator('.reader__picture')).toBeVisible({ timeout: 30_000 })
  await readSettledWidth(first)
  return first
}
