import { test, expect } from '../../support/test.js'

test.describe('Vault Navigation and Quick Open Palette', () => {
  test('navigates to primary entrypoint note via first command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // 2. Search and execute "First" (Go to start note)
    await paletteInput.fill('Go to start note')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 3. Verify opening note is displayed in workspace
    await expect(
      page
        .locator('.note')
        .or(page.getByRole('group', { name: 'Neighbourhood' }))
        .first(),
    ).toBeVisible({ timeout: 10_000 })
  })

  test('switches between notes rapidly via goto quick open palette', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open palette and search Goto command
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Go to a note')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 2. Palette prompts for note name
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Entropy')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 3. Verify Entropy note tab is active
    const entropyTab = page.getByRole('tab', { name: /Entropy/i }).first()
    await expect(entropyTab).toBeVisible({ timeout: 10_000 })
    await expect(entropyTab).toHaveAttribute('aria-selected', 'true')
  })

  test('reveals active note in file tree sidebar via reveal command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open Entropy.md note
    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // 2. Switch away from Files tab to Agent or another panel
    await page.getByText('Agent').first().click()
    await expect(page.getByRole('tab', { name: /^Files/ })).not.toHaveAttribute(
      'aria-selected',
      'true',
    )

    // 3. Run Reveal command (Show in files)
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Show in files')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 4. Verify Files tree is opened and revealed
    const revealedRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(revealedRow).toBeVisible({ timeout: 10_000 })
  })
})
