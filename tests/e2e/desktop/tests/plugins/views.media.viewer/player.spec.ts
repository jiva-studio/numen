import { test, expect, type Page } from '../../support/test.js'
import {
  getPaletteField,
  openWorkspace,
  runCommand,
  runCommandOverItem,
} from '../../support/workspace.js'
import { deleteVaultFiles, listPluginFiles } from '../../support/media.js'

/** The plugin that imports links, and keeps their copies in its folder of the vault. */
const IMPORTER = 'md.numen.tools.media.importer'

/** A caption the mock downloader of `-debug-mock-ml` publishes for every video. */
const CAPTION = 'The mill pond fills behind the gate.'

const listCopies = (): string[] => listPluginFiles(IMPORTER).filter((name) => name.endsWith('.wav'))

const getLinkNote = (id: string): string => `https---www.youtube.com-watch-v=${id}.md`

const importAddress = async (page: Page, address: string, tab: RegExp): Promise<void> => {
  await runCommand(page, 'Import link', /^Import link/)
  await getPaletteField(page).fill(address)
  await page.keyboard.press('Enter')
  await expect(page.getByRole('tab', { name: tab, selected: true })).toBeVisible({
    timeout: 30_000,
  })
}

const downloadText = (page: Page): Promise<void> =>
  runCommand(page, 'Download captions', /^Download captions$/)

const downloadVideo = (page: Page): Promise<void> =>
  runCommandOverItem(page, 'Download media', /^Download media$/, /^Video$/)

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

  test('Delete media takes the copy off the disk, and Download media brings it back', async ({
    page,
  }) => {
    made.push(getLinkNote('tidalmill03'))
    await openWorkspace(page)
    await importAddress(page, 'https://www.youtube.com/watch?v=tidalmill03', /tidalmill03/)
    await downloadText(page)
    await downloadVideo(page)
    const tab = page.getByRole('tabpanel', { name: /tidalmill03/ })
    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    const copies = listCopies()

    await runCommand(page, 'Delete media', /^Delete media$/)

    await expect(tab.getByRole('button', { name: /^Play/ })).toHaveCount(0)
    await expect.poll(() => listCopies()).toHaveLength(copies.length - 1)
    await expect(tab.getByText(CAPTION)).toBeVisible()

    await runCommandOverItem(page, 'Download media', /^Download media$/, /^Video$/)

    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    await expect.poll(() => listCopies()).toHaveLength(copies.length)
  })
})
