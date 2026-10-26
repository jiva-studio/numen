import { test, expect } from 'playwright/test'

test.describe('Deck Management and Scheduling Interactions', () => {
  test('opens deck schedule preset selector menu', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open cards folder and Animals.md deck
    await page.getByRole('tab', { name: /^Files/ }).click()
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')

    const deckRow = page.locator('[data-tree-row="cards/Animals.md"]')
    await expect(deckRow).toBeVisible()
    await deckRow.press('Enter')

    await expect(page.locator('.deck')).toBeVisible({ timeout: 10_000 })

    // 2. Click schedule preset choice button
    const scheduleBtn = page.locator('.deck-tab__choice').first()
    await expect(scheduleBtn).toBeVisible({ timeout: 5_000 })
    await scheduleBtn.click()

    // 3. Verify preset menu appears
    const menu = page.locator('[role="menu"]').or(page.locator('.menu')).first()
    await expect(menu).toBeVisible({ timeout: 5_000 })

    // 4. Press Escape to dismiss menu
    await page.keyboard.press('Escape')
    await expect(menu).not.toBeVisible({ timeout: 5_000 })
  })

  test('displays existing flashcards and fields in deck grid', async ({ page }) => {
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

    // Verify card tiles are rendered
    const cards = page.locator('.card')
    await expect(cards.first()).toBeVisible({ timeout: 10_000 })
    const count = await cards.count()
    expect(count).toBeGreaterThanOrEqual(1)

    // Verify card has value containers
    const firstCard = cards.first()
    await expect(
      firstCard.locator('[data-card-value]').or(firstCard.locator('p, textarea, input')).first(),
    ).toBeVisible()
  })

  test('adds new blank card to deck via stencil selector', async ({ page }) => {
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

    // Click Add a card plus button
    const addCardBtn = page.getByRole('button', { name: 'Add a card' }).first()
    await expect(addCardBtn).toBeVisible({ timeout: 5_000 })
    await addCardBtn.click()

    // Choose first stencil
    const cutOption = page.locator('[data-cut]').first()
    await expect(cutOption).toBeVisible({ timeout: 5_000 })
    await cutOption.click()

    // Verify card count incremented
    await expect(cards).toHaveCount(initialCount + 1, { timeout: 10_000 })
  })

  test('adds new section heading to deck', async ({ page }) => {
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

    const initialSections = await page.locator('.section-heading').count()

    // Click Add section button
    const addSectionBtn = page.locator('[data-add-section]')
    await expect(addSectionBtn).toBeVisible()
    await addSectionBtn.click()

    // Verify section headings count increased
    await expect(page.locator('.section-heading')).toHaveCount(initialSections + 1, {
      timeout: 10_000,
    })
  })
})
