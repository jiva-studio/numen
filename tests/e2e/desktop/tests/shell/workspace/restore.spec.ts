import { test, expect, readSavedStates, type Page } from '../../support/test.js'
import {
  openVaultFile,
  openWorkspace,
  runCommand,
} from '../../support/workspace.js'

const getPageField = (page: Page, name: RegExp) =>
  page.getByRole('tabpanel', { name }).getByRole('spinbutton', { name: 'Page' })

const expectSavedState = async (page: Page, path: string, text: string): Promise<void> => {
  await expect.poll(async () => (await readSavedStates(page.request)).get(path)).toContain(text)
}

const PDF_READER_ID = 'md.numen.readers.pdf'

const setPdfReaderEnabled = async (page: Page, isEnabled: boolean): Promise<void> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  const toggle = settings.getByRole('switch', { name: /Enabled PDF Reader/ })
  if (!(await toggle.isVisible())) {
    await settings.getByRole('button', { name: /PDF Reader/ }).click()
  }
  // The switch moves when the core has answered its write, so the click is not waited for to change it.
  if ((await toggle.isChecked()) !== isEnabled) await toggle.click()
  await expect(toggle).toBeChecked({ checked: isEnabled })
}

test.describe('The tabs of the window after a restart', () => {
  test.describe.configure({ timeout: 120_000 })

  test.afterEach(async ({ page }) => {
    const answer = await page.request.post('/numen.v1.SettingsService/WriteSettings', {
      data: { settings: [{ at: [PDF_READER_ID, 'enabled'], value: 'true' }] },
      headers: { 'Content-Type': 'application/json' },
    })
    expect(answer.ok()).toBe(true)
  })

  test('show a placeholder for a tab whose plugin is disabled, and the tab once it is enabled', async ({
    page,
  }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'shore.pdf')
    await tab.getByRole('button', { name: 'Next page' }).click()
    await expect(getPageField(page, /shore/i)).toHaveValue('2')
    await expectSavedState(page, 'assets/shore.pdf', '"page":2')

    await setPdfReaderEnabled(page, false)
    await page.getByRole('tab', { name: /shore/i }).click()
    await expect(page.getByRole('status')).toContainText('Plugin disabled')

    await setPdfReaderEnabled(page, true)
    await page.getByRole('tab', { name: /shore/i }).click()
    await expect(getPageField(page, /shore/i)).toHaveValue('2')
  })
})
