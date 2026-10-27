import { test, expect } from '../../support/test.js'

test.describe('Flashcard Decks and Card Lifecycle', () => {
  test('creates new flashcard deck file via command palette', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open command palette and execute New deck
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('New deck')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 2. Input deck name
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('CosmologyDeck')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 3. Verify newly created deck tab and editor render
    const deckTab = page.getByRole('tab', { name: /CosmologyDeck/i }).first()
    await expect(deckTab).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.deck').first()).toBeVisible({ timeout: 10_000 })
  })

  test('appends new card to deck via card model template selector', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open cards folder and Animals.md
    await page.getByRole('tab', { name: /^Files/ }).click()
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')

    const deckRow = page.locator('[data-tree-row="cards/Animals.md"]')
    await expect(deckRow).toBeVisible()
    await deckRow.press('Enter')

    await expect(page.locator('.deck')).toBeVisible({ timeout: 10_000 })

    const cards = page.locator('.card')
    await expect(cards.first()).toBeVisible({ timeout: 10_000 })
    const initialCount = await cards.count()

    // 2. Click Add a card plus button
    const addCardBtn = page.getByRole('button', { name: 'Add a card' }).first()
    await expect(addCardBtn).toBeVisible({ timeout: 5_000 })
    await addCardBtn.click()

    // 3. Card model options appear
    const cutOption = page.locator('[data-cut]').first()
    await expect(cutOption).toBeVisible({ timeout: 5_000 })
    await cutOption.click()

    // 4. Verify card count incremented
    await expect(cards).toHaveCount(initialCount + 1, { timeout: 10_000 })
  })

  test('deletes card from flashcard deck', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')

    const deckRow = page.locator('[data-tree-row="cards/Animals.md"]')
    await expect(deckRow).toBeVisible()
    await deckRow.press('Enter')

    await expect(page.locator('.deck')).toBeVisible({ timeout: 10_000 })

    const cards = page.locator('.card')
    await expect(cards.first()).toBeVisible({ timeout: 10_000 })
    const initialCount = await cards.count()

    // Find first card's remove button
    const removeBtn = page.locator('.card .remove-button').first()
    await expect(removeBtn).toBeVisible({ timeout: 5_000 })
    await removeBtn.click()

    // Verify card count decremented
    await expect(cards).toHaveCount(initialCount - 1, { timeout: 10_000 })
  })
})
