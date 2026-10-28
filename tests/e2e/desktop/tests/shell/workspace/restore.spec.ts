import { test, expect, type Page } from '../../support/test.js'
import {
  openVaultFile,
  openWorkspace,
  runCommand,
  waitForPassage,
} from '../../support/workspace.js'

/** The page field of the PDF reader in a panel. */
const getPageField = (page: Page, name: RegExp) =>
  page.getByRole('tabpanel', { name }).getByRole('spinbutton', { name: 'Page' })

/** Turns the PDF reader plugin on or off in the settings. */
const setPdfReaderEnabled = async (page: Page, isEnabled: boolean): Promise<void> => {
  await runCommand(page, 'Settings', /^Settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  const toggle = settings.getByRole('switch', { name: /Enabled PDF Reader/ })
  if (!(await toggle.isVisible())) {
    await settings.getByRole('button', { name: /PDF Reader/ }).click()
  }
  await toggle.setChecked(isEnabled)
  await expect(toggle).toBeChecked({ checked: isEnabled })
}

test.describe('The tabs of the window after a restart', () => {
  test.describe.configure({ timeout: 120_000 })

  test('open a PDF at the page it was left at', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'shore.pdf')
    await tab.getByRole('button', { name: 'Next page' }).click()
    await expect(getPageField(page, /shore/i)).toHaveValue('2')

    await page.reload()

    await expect(getPageField(page, /shore/i)).toHaveValue('2')
  })

  test('open an EPUB at the place it was left at', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'sample.epub')
    await expect(tab.locator('.book-tab__reading')).toBeVisible()
    await tab.locator('.book-tab').press('PageDown')
    const textBefore = await tab.locator('.book-tab__reading').innerText()

    await page.reload()

    const panel = page.getByRole('tabpanel', { name: /sample/i })
    await expect(panel.locator('.book-tab__reading')).toHaveText(textBefore, { useInnerText: true })
  })

  test('open a recording at the time it was left at', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'talk.mp3')
    const player = tab.getByRole('group', { name: 'Recording' })
    await expect(player).toContainText('3:07')
    await player.getByRole('slider').fill('120000')
    await expect(player).toContainText('2:00')

    await page.reload()

    const panel = page.getByRole('tabpanel', { name: /talk/i }).getByRole('group', {
      name: 'Recording',
    })
    await expect(panel).toContainText('2:00')
  })

  test('show a placeholder for a tab whose plugin is disabled, and the tab once it is enabled', async ({
    page,
  }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'shore.pdf')
    await tab.getByRole('button', { name: 'Next page' }).click()
    await expect(getPageField(page, /shore/i)).toHaveValue('2')

    await setPdfReaderEnabled(page, false)
    await page.getByRole('tab', { name: /shore/i }).click()
    await expect(page.getByRole('status')).toContainText('Plugin disabled')

    await setPdfReaderEnabled(page, true)
    await page.getByRole('tab', { name: /shore/i }).click()
    await expect(getPageField(page, /shore/i)).toHaveValue('2')
  })
})

test.describe('A search hit in a PDF', () => {
  test.describe.configure({ timeout: 120_000 })

  test('opens the PDF at the page of the hit', async ({ page }) => {
    await openWorkspace(page)

    const passage = await waitForPassage(page, 'heron wades', 'assets/shore.pdf')
    await passage.click()

    await expect(getPageField(page, /shore/i)).toHaveValue('2')
  })
})
