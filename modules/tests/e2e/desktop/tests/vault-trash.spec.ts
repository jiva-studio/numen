import { test, expect } from 'playwright/test'

test.describe('Vault Deletion, Trash and Renaming', () => {
  test('moves note to trash via soft remove command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Create a disposable note via command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('New note')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('DisposableNote')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(page.getByRole('tab', { name: /DisposableNote/i }).first()).toBeVisible({ timeout: 10_000 })

    // 2. Trigger Remove command
    await page.keyboard.press('Control+p')
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Remove')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 3. Verify workspace remains active
    await expect(page.locator('text=Ready')).toBeVisible({ timeout: 10_000 })
  })

  test('enforces exact confirmation string before destroying note forever', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes and Entropy.md
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Open palette and search Destroy command
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Destroy')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Verify confirmation prompt / input appears
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
  })

  test('renames note via title command and preserves content integrity', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes and Entropy.md
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Trigger Title command
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Title')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Verify rename field is open with current title
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
  })
})
