import { test, expect } from 'playwright/test'
import { openVaultFile, openWorkspace, runCommand, waitForPassage } from '../support/workspace.js'

/** What the mock transcriber of `-debug-mock-ml` hears in every recording. */
const MOCK_TRANSCRIPT = 'Mock transcribed text for automated test.'

// A recording is transcribed when a person asks, and each test after the one that asks reads the
// transcript it made.
test.describe.configure({ mode: 'serial' })

test.describe('A recording', () => {
  test('plays: its length is known and the time runs past 0:00', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'lecture.wav')

    await expect(tab).toContainText('0:06')
    await tab.getByRole('button', { name: /^Play/ }).click()

    await expect(tab.getByRole('button', { name: /^Pause/ })).toBeVisible()
    await expect(tab).toContainText(/0:0[1-5]/)
  })

  test('is transcribed when a person asks, and its transcript is shown under the player', async ({
    page,
  }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'lecture.wav')
    await runCommand(page, 'Recognise', /Audio\/Video Transcription/)

    await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible({ timeout: 30_000 })
  })

  test('keeps an edit of its transcript, and the search finds the edit', async ({ page }) => {
    await openWorkspace(page)
    let tab = await openVaultFile(page, 'assets', 'lecture.wav')
    const transcript = tab.getByRole('textbox').filter({ hasText: MOCK_TRANSCRIPT })
    await expect(transcript).toBeVisible({ timeout: 30_000 })

    await transcript.click()
    await page.keyboard.press('Control+End')
    await page.keyboard.type(' The lecturer paused at the blackboard.')
    const writeResponse = page.waitForResponse((response) =>
      response.url().includes('/numen.v1.TextService/CreateTextEdit'),
    )
    await page.keyboard.press('Control+s')
    expect((await writeResponse).ok()).toBe(true)

    await page.reload()
    await openWorkspace(page)
    tab = await openVaultFile(page, 'assets', 'lecture.wav')
    await expect(tab.getByText(/The lecturer paused at the blackboard\./)).toBeVisible()

    const passage = await waitForPassage(page, 'lecturer paused', 'assets/lecture.wav')
    await expect(passage).toContainText('blackboard')
  })

  test('is found through the search by the words of its transcript', async ({ page }) => {
    await openWorkspace(page)

    const passage = await waitForPassage(page, 'Mock transcribed', 'assets/lecture.wav')
    await passage.click()

    const tab = page.getByRole('tabpanel', { name: /lecture/i })
    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible()
    await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible()
  })
})
