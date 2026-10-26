import { test, expect, type Page } from 'playwright/test'
import {
  findPassages,
  openVaultFile,
  openWorkspace,
  runCommand,
  waitForPassage,
} from '../support/workspace.js'

/** The words of the page as pixels, which only a recognition reads. */
const SCANNED_WORDS = 'rotor spinning in a vacuum'
/** The words of the text layer the scanner wrote into the file. */
const LAYER_WORDS = 'shipping manifest'
const PATH = 'assets/flywheel.pdf'
/** What the mock proofreader adds to the end of every line it puts right. */
const PROOFREAD_MARK = 'proofread'

const setOcrEnabled = async (page: Page, isEnabled: boolean): Promise<void> => {
  await runCommand(page, 'Settings', /^Settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /Document OCR/ }).click()
  const toggle = settings.getByRole('switch', { name: /Enabled Document OCR/ })
  await toggle.setChecked(isEnabled)
  await expect(toggle).toBeChecked({ checked: isEnabled })
}

test.describe('With the OCR plugin disabled', () => {
  test.describe.configure({ timeout: 180_000 })

  test.afterEach(async ({ page }) => {
    await openWorkspace(page)
    await setOcrEnabled(page, true)
  })

  test('the text shown falls back to the text layer of the file', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'flywheel.pdf')
    await waitForPassage(page, LAYER_WORDS, PATH)
    await page.keyboard.press('Escape')

    await runCommand(page, 'Recognise', /^Recognise/)
    await waitForPassage(page, SCANNED_WORDS, PATH)
    await page.keyboard.press('Escape')

    await setOcrEnabled(page, false)

    await waitForPassage(page, LAYER_WORDS, PATH)
    await expect((await findPassages(page, SCANNED_WORDS, PATH)).first()).toBeHidden()
  })
})

test.describe('With the OCR plugin disabled after a proofreading', () => {
  test.describe.configure({ timeout: 180_000 })

  test.afterEach(async ({ page }) => {
    await openWorkspace(page)
    await setOcrEnabled(page, true)
  })

  test('shows the text layer while the plugin is off, and the proofread recognition once it is on', async ({
    page,
  }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'flywheel.pdf')
    await runCommand(page, 'Recognise', /^Recognise/)
    await waitForPassage(page, SCANNED_WORDS, PATH)
    await page.keyboard.press('Escape')

    await runCommand(page, 'Proofread', /^Proofread/)
    await waitForPassage(page, PROOFREAD_MARK, PATH)
    await page.keyboard.press('Escape')

    await setOcrEnabled(page, false)

    await waitForPassage(page, LAYER_WORDS, PATH)
    await expect((await findPassages(page, PROOFREAD_MARK, PATH)).first()).toBeHidden()

    await openWorkspace(page)
    await setOcrEnabled(page, true)

    await waitForPassage(page, PROOFREAD_MARK, PATH)
  })
})
