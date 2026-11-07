import { test, expect, type Page } from '../support/test.js'
import {
  getPaletteField,
  openVaultFile,
  openWorkspace,
  runCommand,
  runCommandOverItem,
  waitForPassage,
} from '../support/workspace.js'

/** Words of the text the mock recogniser of `-debug-mock-ml` reads on every page. */
const MOCK_RECOGNISED_WORDS = 'upper reservoir'
/**
 * The transcript the mock transcriber of `-debug-mock-ml` produces for every
 * recording.
 */
const MOCK_TRANSCRIPT = /Mock transcribed text/
/** What the mock proofreader of `-debug-mock-ml` adds to the end of every line it puts right. */
const PROOFREAD_MARK = 'proofread'

/** Closes the palette: Escape leaves a pick list for the list of commands, and closes that list. */
const closePalette = async (page: Page): Promise<void> => {
  const field = getPaletteField(page)
  for (let press = 0; press < 2 && (await field.isVisible()); press += 1) {
    await page.keyboard.press('Escape')
  }
  await expect(field).toBeHidden()
}

/**
 * Waits until the pick list of deleting a proofreading offers the version a proofreading made of
 * the open file beside the one it was made from.
 */
const expectProofreadVersion = async (page: Page): Promise<void> => {
  await expect(async () => {
    await closePalette(page)
    await runCommand(page, 'Delete text', /^Delete text/)
    await expect(page.getByRole('option')).toHaveCount(2, { timeout: 2_000 })
  }).toPass({ timeout: 90_000 })
  await closePalette(page)
}

/** Proofreads the live text version of the open file, picked from the versions the command lists. */
const proofreadLiveVersion = async (page: Page): Promise<void> => {
  await runCommandOverItem(page, 'Proofread', /^Proofread/, /\(live\)$/)
}

test.describe('Proofreading', () => {
  test.describe.configure({ timeout: 180_000 })

  test('puts the recognised text of a scan right', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'pumped-storage.pdf')
    await runCommandOverItem(page, 'Create text', /^Create text$/, /^Recognize text$/)
    await waitForPassage(page, MOCK_RECOGNISED_WORDS, 'assets/pumped-storage.pdf')
    await page.keyboard.press('Escape')

    await proofreadLiveVersion(page)

    await expectProofreadVersion(page)
    await waitForPassage(page, PROOFREAD_MARK, 'assets/pumped-storage.pdf')
    await waitForPassage(page, MOCK_RECOGNISED_WORDS, 'assets/pumped-storage.pdf')
  })

  test('puts the transcript of a recording right', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'lecture.wav')
    await runCommand(page, 'Transcribe', /^Transcribe/)
    await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible({ timeout: 30_000 })

    await proofreadLiveVersion(page)

    await expectProofreadVersion(page)
    await expect(tab.getByText(/Mock transcribed text.*\[proofread\]/)).toBeVisible()
    await waitForPassage(page, PROOFREAD_MARK, 'assets/lecture.wav')
  })
})
