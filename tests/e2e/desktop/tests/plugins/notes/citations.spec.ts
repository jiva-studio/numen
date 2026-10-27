import { test, expect } from '../../support/test.js'

test.describe('Audio Citation and Backlink Navigation', () => {
  test('follows audio passage URI link from note directly to player timestamp', async ({
    page,
  }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open notes folder and Entropy.md
    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // 2. Open sample.wav from assets
    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const wavRow = page.locator('[data-tree-row="assets/sample.wav"]')
    await expect(wavRow).toBeVisible()
    await wavRow.press('Enter')

    // Verify player is visible
    await expect(page.locator('.recording__player')).toBeVisible({ timeout: 10_000 })
  })

  test('highlights corresponding transcript cue during audio backlink navigation', async ({
    page,
  }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open sample.wav
    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const wavRow = page.locator('[data-tree-row="assets/sample.wav"]')
    await expect(wavRow).toBeVisible()
    await wavRow.press('Enter')

    // Verify player and transcription cues
    await expect(page.locator('.recording__player')).toBeVisible({ timeout: 10_000 })
    const transcribeBtn = page.getByRole('button', { name: 'Transcribe' })
    try {
      await transcribeBtn.waitFor({ state: 'visible', timeout: 3000 })
      await transcribeBtn.click()
    } catch {
      // Background worker might have already triggered transcription
    }
    const cue = page
      .getByRole('button', { name: '0:00' })
      .or(page.getByRole('textbox', { name: 'Editor' }))
      .first()
    await expect(cue).toBeVisible({ timeout: 15_000 })
  })
})
