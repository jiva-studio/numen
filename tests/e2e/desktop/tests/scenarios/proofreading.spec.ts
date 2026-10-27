import { test, expect, type Page } from 'playwright/test'
import { openVaultFile, openWorkspace, runCommand, waitForPassage } from '../support/workspace.js'

/** Words of the text the mock recogniser of `-debug-mock-ml` reads on every page. */
const MOCK_RECOGNISED_WORDS = 'upper reservoir'
/** What the mock transcriber of `-debug-mock-ml` hears in every recording. */
const MOCK_TRANSCRIPT = /Mock transcribed text/
/** What the mock proofreader of `-debug-mock-ml` adds to the end of every line it puts right. */
const PROOFREAD_MARK = 'proofread'

/** Waits until the palette offers deleting the version a proofreading made of the open file. */
const expectProofreadVersion = async (page: Page): Promise<void> => {
  await expect(async () => {
    await page.keyboard.press('Escape')
    await page.keyboard.press('Control+p')
    await page.getByRole('combobox', { name: 'Search the vault' }).fill('Delete')
    await expect(
      page.getByRole('option', { name: /^(Delete|Remove)\b.*\b(proofread|corrected)/i }).first(),
    ).toBeVisible({ timeout: 2_000 })
  }).toPass({ timeout: 90_000 })
  await page.keyboard.press('Escape')
}

test.describe('Proofreading', () => {
  test.describe.configure({ timeout: 180_000 })

  test('puts the recognised text of a scan right', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'pumped-storage.pdf')
    await runCommand(page, 'Recognise', /^Recognise/)
    await waitForPassage(page, MOCK_RECOGNISED_WORDS, 'assets/pumped-storage.pdf')
    await page.keyboard.press('Escape')

    await runCommand(page, 'Proofread', /^Proofread/)

    await expectProofreadVersion(page)
    await waitForPassage(page, PROOFREAD_MARK, 'assets/pumped-storage.pdf')
    await waitForPassage(page, MOCK_RECOGNISED_WORDS, 'assets/pumped-storage.pdf')
  })

  test('puts the transcript of a recording right', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'lecture.wav')
    await runCommand(page, 'Transcribe', /^Transcribe with Audio\/Video Transcription/)
    await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible({ timeout: 30_000 })

    await runCommand(page, 'Proofread', /^Proofread/)

    await expectProofreadVersion(page)
    await expect(tab.getByText(/Mock transcribed text.*\[proofread\]/)).toBeVisible()
    await waitForPassage(page, PROOFREAD_MARK, 'assets/lecture.wav')
  })
})
