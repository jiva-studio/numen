import { test, expect } from 'playwright/test'

test.describe('Audio Transcription Pipeline', () => {
  test('opens audio recording and executes transcription with mock ML', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Verify sample.wav is listed
    const wavRow = page.locator('[data-tree-row="assets/sample.wav"]')
    await expect(wavRow).toBeVisible()

    // Activate sample.wav to open recording tab
    await wavRow.press('Enter')

    // Verify recording tab is opened
    await expect(page.locator('.recording__player')).toBeVisible()

    // If transcribe button is visible, click it; background worker may auto-transcribe with mock ML
    const transcribeBtn = page.getByRole('button', { name: 'Transcribe' })
    try {
      await transcribeBtn.waitFor({ state: 'visible', timeout: 3000 })
      await transcribeBtn.click()
    } catch {
      // Background worker might have already triggered transcription
    }

    // Mock ML transcribes the audio and generates transcript text
    const editor = page.locator('.transcript__text')
    await expect(editor).toBeVisible({ timeout: 15_000 })
  })

  test('opens mp3 audio file and renders media recording player', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Verify sample.mp3 is listed
    const mp3Row = page.locator('[data-tree-row="assets/sample.mp3"]')
    await expect(mp3Row).toBeVisible()

    // Activate sample.mp3
    await mp3Row.press('Enter')

    // Verify recording player or recording note rendered
    await expect(
      page.locator('.recording__player').or(page.locator('.recording__note')),
    ).toBeVisible({ timeout: 10_000 })
  })
})
