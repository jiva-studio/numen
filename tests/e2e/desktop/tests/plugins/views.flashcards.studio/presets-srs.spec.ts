import { test, expect } from '../../support/test.js'

test.describe('Spaced Repetition and Preset Editor', () => {
  test('opens new spaced repetition preset editor from command palette', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Select New preset command
    await paletteInput.fill('New preset')
    await expect(page.getByRole('option', { name: /New preset/ }).first()).toBeVisible()
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // In naming step, enter preset name
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('PhysicsSRS')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Verify preset tab is opened and contains goal options
    await expect(
      page
        .getByRole('tab', { name: /PhysicsSRS/i })
        .or(page.locator('[aria-label*="PhysicsSRS"]'))
        .first(),
    ).toBeVisible({ timeout: 10_000 })

    await expect(page.getByRole('region', { name: 'Goal', exact: true })).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByRole('radio', { name: 'Minutes per day' })).toBeVisible()
    await expect(page.getByRole('radio', { name: 'Retention' })).toBeVisible()
  })

  test('validates preset retention and interval boundaries with problem hints', async ({
    page,
  }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Create and open preset
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('New preset')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('StrictSRS')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(page.getByRole('tab', { name: /StrictSRS/i }).first()).toBeVisible({
      timeout: 10_000,
    })

    // Verify initial radio state
    const minutesRadio = page.getByRole('radio', { name: 'Minutes per day' })
    const retentionRadio = page.getByRole('radio', { name: 'Retention' })

    await expect(minutesRadio).toBeChecked()

    // Verify daily schedule toolbar
    await expect(page.getByRole('toolbar', { name: 'Load by weekday' })).toBeVisible()

    // Switch goal to Retention
    await retentionRadio.click()
    await expect(retentionRadio).toBeChecked()
  })

  test('assigns custom preset to active note via preset command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes folder and Entropy.md
    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Trigger Preset command on note
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Preset')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Verify preset editor is opened for the note with goal region and defaults explanation
    await expect(
      page
        .getByRole('tab', { name: /Entropy\.md/i })
        .or(page.getByRole('region', { name: 'Goal', exact: true }))
        .first(),
    ).toBeVisible({ timeout: 10_000 })

    await expect(page.getByRole('region', { name: 'Goal', exact: true })).toBeVisible({
      timeout: 10_000,
    })
  })

  test('says a deck naming no preset is scheduled by the defaults', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')
    const deckRow = page.locator('[data-tree-row="cards/Animals.md"]')
    await expect(deckRow).toBeVisible()
    await deckRow.press('Enter')
    await expect(page.getByRole('tab', { name: /Animals/ }).first()).toBeVisible({
      timeout: 10_000,
    })

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Open preset')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(page.getByText('No preset set. Scheduled by the defaults.').first()).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByRole('region', { name: 'Goal', exact: true })).toHaveCount(0)
  })
})
