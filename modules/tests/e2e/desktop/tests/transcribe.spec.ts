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

    // Find and click the Transcribe button
    const transcribeBtn = page.locator('.transcript__ask')
    await expect(transcribeBtn).toBeVisible()
    await transcribeBtn.click()

    // Mock ML transcribes the audio and generates transcript text
    const editor = page.locator('.transcript__text')
    await expect(editor).toBeVisible({ timeout: 10_000 })
  })
})
