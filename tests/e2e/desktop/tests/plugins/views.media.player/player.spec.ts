import { test, expect, type Page } from '../../support/test.js'
import { getPaletteField, openWorkspace, runCommand } from '../../support/workspace.js'
import { deleteVaultFiles, listPluginFiles, restartPlugin } from '../../support/media.js'

/** The plugin that imports links, and keeps their copies in its folder of the vault. */
const IMPORTER = 'md.numen.tools.media.importer'

/** A caption the mock downloader of `-debug-mock-ml` publishes for every video. */
const CAPTION = 'The mill pond fills behind the gate.'

const listCopies = (): string[] => listPluginFiles(IMPORTER).filter((name) => name.endsWith('.wav'))

const getLinkNote = (id: string): string => `https---www.youtube.com-watch-v=${id}.md`

const importAddress = async (page: Page, address: string): Promise<void> => {
  await runCommand(page, 'Import from URL', /^Import from URL/)
  await getPaletteField(page).fill(address)
  await page.keyboard.press('Enter')
}

test.describe('a link to a video', () => {
  test.describe.configure({ timeout: 120_000 })

  const made: string[] = []

  test.beforeEach(async ({ page }) => {
    // A link without a copy embeds the site's player, which is not reached from a test.
    await page.route(/youtube-nocookie\.com/, (route) => route.abort())
  })

  test.afterEach(() => {
    deleteVaultFiles(...made.splice(0))
  })

  test('Delete downloaded video takes the copy off the disk, and Download video brings it back', async ({
    page,
  }) => {
    made.push(getLinkNote('tidalmill03'))
    await openWorkspace(page)
    await importAddress(page, 'https://www.youtube.com/watch?v=tidalmill03')
    const tab = page.getByRole('tabpanel').filter({ hasText: CAPTION })
    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    const copies = listCopies()

    await tab.getByRole('button', { name: 'Delete downloaded video' }).click()

    await expect(tab.getByRole('button', { name: /^Play/ })).toHaveCount(0)
    await expect(tab.getByRole('button', { name: 'Download video' })).toBeVisible()
    await expect.poll(() => listCopies()).toHaveLength(copies.length - 1)
    await expect(tab.getByText(CAPTION)).toBeVisible()

    await tab.getByRole('button', { name: 'Download video' }).click()

    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    await expect.poll(() => listCopies()).toHaveLength(copies.length)
  })
})

test.describe('a link to an address that answers 404', () => {
  const made: string[] = []

  // The failed fetch stays in the list of tasks until the importer stops.
  test.afterEach(async ({ request }) => {
    deleteVaultFiles(...made.splice(0))
    await restartPlugin(request, IMPORTER)
  })
})
