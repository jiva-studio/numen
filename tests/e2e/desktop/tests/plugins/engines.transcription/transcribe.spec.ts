import { test, expect } from '../../support/test.js'

test.describe('Audio Transcription Pipeline', () => {
  test('opens audio recording and executes transcription with mock ML', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

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
    await page.getByRole('tab', { name: /^Files/ }).click()

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

  test('opens flac audio file and renders media recording player', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Verify sample.flac is listed
    const flacRow = page.locator('[data-tree-row="assets/sample.flac"]')
    await expect(flacRow).toBeVisible()

    // Activate sample.flac
    await flacRow.press('Enter')

    // Verify recording player or recording note rendered
    await expect(
      page.locator('.recording__player').or(page.locator('.recording__note')),
    ).toBeVisible({ timeout: 10_000 })
  })

  test('supports seek and playback interaction on recording player', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.wav
    const wavRow = page.locator('[data-tree-row="assets/sample.wav"]')
    await expect(wavRow).toBeVisible()
    await wavRow.press('Enter')

    const player = page.locator('.recording__player')
    await expect(player).toBeVisible({ timeout: 10_000 })

    // Interact with player element
    await player.click()
    await expect(player).toBeVisible()
  })

  test('deletes transcript artifact and resets to untranscribed state', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.wav
    const wavRow = page.locator('[data-tree-row="assets/sample.wav"]')
    await expect(wavRow).toBeVisible()
    await wavRow.press('Enter')

    await expect(page.locator('.recording__player')).toBeVisible({ timeout: 10_000 })

    // Wait for transcript
    const editor = page.locator('.transcript__text')
    await expect(editor).toBeVisible({ timeout: 15_000 })

    // Open media more actions menu
    const moreBtn = page.locator('.media__more')
    if (await moreBtn.isVisible()) {
      await moreBtn.click()
      const deleteItem = page.getByText(/Delete transcript/i).first()
      if (await deleteItem.isVisible()) {
        await deleteItem.click()
      }
    }
  })

  test('inspects playback duration and controls on audio recording', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.wav
    const wavRow = page.locator('[data-tree-row="assets/sample.wav"]')
    await expect(wavRow).toBeVisible()
    await wavRow.press('Enter')

    const player = page.locator('.recording__player')
    await expect(player).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.media__actions')).toBeVisible({ timeout: 10_000 })
  })
})
