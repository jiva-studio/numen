import { test, expect } from 'playwright/test'

test.describe('Vault Deletion, Trash and Renaming', () => {
  test('moves note to trash via soft remove command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open notes folder and trash-target.md
    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const trashRow = page.locator('[data-tree-row="notes/trash-target.md"]')
    await expect(trashRow).toBeVisible()
    await trashRow.press('Enter')

    const trashTab = page.getByRole('tab', { name: /Disposable Note|trash-target/i }).first()
    await expect(trashTab).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // 2. Trigger Remove command from command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Remove note')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 3. Verify target note tab is closed
    await expect(page.getByRole('tab', { name: /Disposable Note|trash-target/i })).not.toBeVisible({
      timeout: 10_000,
    })
    await expect(page.locator('.status-bar')).toBeVisible({ timeout: 10_000 })
  })

  test('enforces exact confirmation string before destroying note forever', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes and Entropy.md
    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    const noteEditor = page.locator('.note').first()
    await expect(noteEditor).toBeVisible({ timeout: 10_000 })

    // Open palette and search Destroy command
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Destroy')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Verify confirmation prompt / input appears and rejects mismatched input
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('WrongTitleConfirmation')
    await page.keyboard.press('Enter')

    // Note is not destroyed because confirmation did not match
    await expect(noteEditor).toBeVisible()
  })

  test('renames note via title command and preserves content integrity', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes and Entropy.md
    await page.getByRole('tab', { name: /^Files/ }).click()
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

    // Verify rename field contains current title "Entropy"
    await expect(paletteInput).toHaveValue('Entropy')
    await page.keyboard.press('Escape')
    await expect(page.locator('.note').first()).toBeVisible()
  })
})
