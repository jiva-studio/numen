import { test, expect, readActivePath, readSavedStates, type Page } from '../../support/test.js'
import {
  openVaultFile,
  openWorkspace,
  runCommand,
  waitForPassage,
} from '../../support/workspace.js'

declare global {
  interface Window {
    placeholderTexts: string[]
  }
}

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

  test('open a PDF at the page it was left at', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'shore.pdf')
    await tab.getByRole('button', { name: 'Next page' }).click()
    await expect(getPageField(page, /shore/i)).toHaveValue('2')
    await expectSavedState(page, 'assets/shore.pdf', '"page":2')

    await page.reload()

    await expect(getPageField(page, /shore/i)).toHaveValue('2')
  })

  test('open an EPUB at the place it was left at', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'sample.epub')
    const bookText = tab.locator('.book-tab__reading')
    await expect(bookText).toContainText(/\d+ of \d+/)
    await tab.locator('.book-tab').press('PageDown')
    await expect(bookText).toContainText(/\d+ of \d+/)
    const textBefore = await bookText.innerText()

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
    await expectSavedState(page, 'assets/talk.mp3', '"t":[120,')

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
    await expectSavedState(page, 'assets/shore.pdf', '"page":2')

    await setPdfReaderEnabled(page, false)
    await page.getByRole('tab', { name: /shore/i }).click()
    await expect(page.getByRole('status')).toContainText('Plugin disabled')

    await setPdfReaderEnabled(page, true)
    await page.getByRole('tab', { name: /shore/i }).click()
    await expect(getPageField(page, /shore/i)).toHaveValue('2')
  })

  test('show only the disabled text on a restored tab of a disabled plugin, and enable it from there', async ({
    page,
  }) => {
    await page.addInitScript(() => {
      const texts: string[] = []
      window.placeholderTexts = texts
      const record = () => {
        for (const one of document.querySelectorAll('.tab-unavailable h3, .tab-unavailable p')) {
          const text = one.textContent?.trim() ?? ''
          if (text !== '' && !texts.includes(text)) texts.push(text)
        }
      }
      new MutationObserver(record).observe(document, {
        childList: true,
        subtree: true,
        characterData: true,
      })
    })
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'shore.pdf')
    await expect(getPageField(page, /shore/i)).toHaveValue('1')
    await setPdfReaderEnabled(page, false)
    await page.getByRole('tab', { name: /shore/i }).click()
    await expect.poll(() => readActivePath(page.request)).toBe('assets/shore.pdf')

    await page.reload()

    const panel = page.getByRole('tabpanel', { name: /shore/i })
    await expect(panel.getByRole('heading', { name: 'Plugin disabled' })).toBeVisible()
    const placeholderTexts = await page.evaluate(() => window.placeholderTexts)
    expect(placeholderTexts).toStrictEqual([
      'Plugin disabled',
      'The plugin for PDF documents is disabled.',
    ])

    const enable = panel.getByRole('button', { name: 'Enable PDF Reader' })
    await expect(enable).toHaveAttribute('data-slot', 'button')
    await enable.click()

    await expect(getPageField(page, /shore/i)).toHaveValue('1')
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
