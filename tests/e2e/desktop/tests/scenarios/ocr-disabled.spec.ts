import { test, expect, type Page } from '../support/test.js'
import {
  findPassages,
  openVaultFile,
  openWorkspace,
  runCommand,
  runCommandOverItem,
  waitForPassage,
} from '../support/workspace.js'

/** Words of the text the mock recogniser of `-debug-mock-ml` reads on every page; the text layer does not hold them. */
const SCANNED_WORDS = 'rotor spinning in a vacuum'
/** The words of the text layer the scanner wrote into the file. */
const LAYER_WORDS = 'shipping manifest'
const PATH = 'assets/flywheel.pdf'
/** What the mock proofreader of `-debug-mock-ml` adds to the end of every line it puts right. */
const PROOFREAD_MARK = 'proofread'

const setOcrEnabled = async (page: Page, isEnabled: boolean): Promise<void> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /Document OCR/ }).click()
  const toggle = settings.getByRole('switch', { name: /Enabled Document OCR/ })
  // The switch moves when the core has answered its write, so the click is not waited for to change it.
  if ((await toggle.isChecked()) !== isEnabled) await toggle.click()
  await expect(toggle).toBeChecked({ checked: isEnabled })
}

test.describe('With the OCR plugin disabled', () => {
  test.describe.configure({ timeout: 180_000 })

  test.afterEach(async ({ page }) => {
    await openWorkspace(page)
    await setOcrEnabled(page, true)
  })

  test('the recognition stays the text shown and found', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'flywheel.pdf')
    await waitForPassage(page, LAYER_WORDS, PATH)
    await page.keyboard.press('Escape')

    await runCommandOverItem(page, 'Create text', /^Create text$/, /^Recognize text$/)
    await waitForPassage(page, SCANNED_WORDS, PATH)
    await page.keyboard.press('Escape')

    await setOcrEnabled(page, false)

    await waitForPassage(page, SCANNED_WORDS, PATH)
    await expect((await findPassages(page, LAYER_WORDS, PATH)).first()).toBeHidden()
  })
})

test.describe('With the OCR plugin disabled after a proofreading', () => {
  test.describe.configure({ timeout: 180_000 })

  test.afterEach(async ({ page }) => {
    await openWorkspace(page)
    await setOcrEnabled(page, true)
  })

  test('keeps the proofread recognition shown and found', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'flywheel.pdf')
    await runCommandOverItem(page, 'Create text', /^Create text$/, /^Recognize text$/)
    await waitForPassage(page, SCANNED_WORDS, PATH)
    await page.keyboard.press('Escape')

    await runCommandOverItem(page, 'Proofread', /^Proofread/, /\(live\)$/)
    await waitForPassage(page, PROOFREAD_MARK, PATH)
    await page.keyboard.press('Escape')

    await setOcrEnabled(page, false)

    await waitForPassage(page, PROOFREAD_MARK, PATH)
    await expect((await findPassages(page, LAYER_WORDS, PATH)).first()).toBeHidden()
  })
})
