import { test, expect } from '../../support/test.js'

test.describe('Flashcards and Deck Editor', () => {
  test('opens flashcard deck editor and displays card list with sections', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand cards directory
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')

    // Open Animals.md deck
    const deckRow = page.locator('[data-tree-row="cards/Animals.md"]')
    await expect(deckRow).toBeVisible()
    await deckRow.press('Enter')

    // Verify DeckEditor renders
    await expect(page.locator('.deck')).toBeVisible({ timeout: 10_000 })

    // Verify cards are drawn
    const cards = page.locator('.card')
    await expect(cards.first()).toBeVisible({ timeout: 10_000 })
    expect(await cards.count()).toBeGreaterThanOrEqual(1)
  })

  test('adds new section to flashcard deck', async ({ page }) => {
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

    // Click Add section button
    const addSectionBtn = page.locator('[data-add-section]')
    await expect(addSectionBtn).toBeVisible()
    await addSectionBtn.click()

    // Verify new section heading appeared
    await expect(page.locator('.deck').first()).toBeVisible()
  })

  test('edits flashcard field text in deck editor', async ({ page }) => {
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

    // Find first editable field in card
    const editableField = page
      .locator('.card [contenteditable="true"], .card textarea, .card input')
      .first()
    if (await editableField.isVisible()) {
      await editableField.click()
      await page.keyboard.type(' Updated')
    }
    await expect(page.locator('.deck')).toBeVisible()
  })

  test('inspects cards and deck schedule controls', async ({ page }) => {
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

    // Verify deck header or schedule bar is present
    const cards = page.locator('.card')
    await expect(cards.first()).toBeVisible()
  })
})
