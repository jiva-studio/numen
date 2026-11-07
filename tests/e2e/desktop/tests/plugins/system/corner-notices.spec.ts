import { observeFor } from '../../support/browser.js'
import { test, expect, type Page } from '../../support/test.js'
import {
  getPaletteField,
  openTaskList,
  openWorkspace,
  runCommand,
} from '../../support/workspace.js'
import { listFilePaths } from '../../support/vault.js'
import {
  deleteVaultFiles,
  getStandInsOrigin,
  restartPlugin,
  writePluginSetting,
} from '../../support/media.js'

const IMPORTER = 'md.numen.tools.media.importer'
const UPDATER = 'md.numen.tools.updater'

/** An address the proxy of `scripts/start-server.mjs` answers 404 for. */
const MISSING_PAGE_ADDRESS = 'http://unreachable.numen.test/tide-tables'

/** Everything the removed corner notices were drawn with: the container, the cards and their classes. */
const OLD_CORNER =
  '.notices, .notices__stack, .notice, article.notice, .notice__work, .notice__away'

/** The corner region as it was announced: a status or alert region holding cards. */
const getOldRegions = (page: Page) =>
  page.locator('[role="status"], [role="alert"]').filter({ has: page.locator('.notice') })

const expectNoCorner = async (page: Page): Promise<void> => {
  await expect(page.locator(OLD_CORNER)).toHaveCount(0)
  await expect(getOldRegions(page)).toHaveCount(0)
}

test.describe('the window shows no corner notices', () => {
  test('after the window opens and the vault is indexed, in words-only mode as it is', async ({
    page,
  }) => {
    await openWorkspace(page)
    await observeFor(page, 1_000)

    await expectNoCorner(page)
    await expect(page.getByText('Indexing the vault…')).toHaveCount(0)
    await expect(page.getByText('Text search only: no embedding model set')).toHaveCount(0)
  })

  test.describe('a plugin action that fails', () => {
    const made: string[] = []

    test.afterEach(async ({ request }) => {
      deleteVaultFiles(...made.splice(0))
      await restartPlugin(request, IMPORTER)
    })

    test('is a failed task, not a notice', async ({ page, request }) => {
      const before = await listFilePaths(request)
      await openWorkspace(page)
      await runCommand(page, 'Import link', /^Import link/)
      await getPaletteField(page).fill(MISSING_PAGE_ADDRESS)
      await page.keyboard.press('Enter')
      // Import link makes the link note alone; its text is fetched by a command over the note.
      await expect(page.getByRole('tab', { name: /tide-tables/, selected: true })).toBeVisible({
        timeout: 30_000,
      })
      made.push(...(await listFilePaths(request)).filter((one) => !before.includes(one)))
      await runCommand(page, 'Download captions', /^Download captions$/)

      // The failed task of the fetch is the one entry of the failure: the window writes no error of
      // its own for a run whose task started.
      const popover = await openTaskList(page)
      const failures = popover
        .locator('.task-item')
        .filter({ hasText: 'tide-tables answered 404 Not Found' })
      await expect(failures).toHaveCount(1)
      await expect(failures).toContainText('Fetching the text of a link')
      await observeFor(page, 1_000)
      await expect(failures).toHaveCount(1)
      await expect(popover).not.toContainText('Numen is not responding')
      await expectNoCorner(page)
    })
  })

  test.describe('a Custom channel with no release feed address', () => {
    let endpoint = ''

    test.beforeEach(async ({ request }) => {
      endpoint = `${await getStandInsOrigin(request)}/latest/latest.json`
    })

    test.afterEach(async ({ request }) => {
      await writePluginSetting(request, UPDATER, 'channel', 'custom')
      await writePluginSetting(request, UPDATER, 'endpoint', endpoint)
    })

    test('is a failed task, not a notice', async ({ page, request }) => {
      await writePluginSetting(request, UPDATER, 'channel', 'custom')
      await writePluginSetting(request, UPDATER, 'endpoint', '')
      await openWorkspace(page)

      const popover = await openTaskList(page)
      await expect(popover).toContainText('release feed address')
      await expectNoCorner(page)
    })
  })
})
