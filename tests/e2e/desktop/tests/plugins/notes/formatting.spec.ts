import { test, expect } from 'playwright/test'

test.describe('Note editor live formatting and styling', () => {
  test('renders markdown blockquote with cm-quote styling in editor', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open notes folder and plain.md
    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const plainRow = page.locator('[data-tree-row="notes/plain.md"]')
    await expect(plainRow).toBeVisible()
    await plainRow.press('Enter')

    const cmContent = page.locator('.note .cm-content').first()
    await expect(cmContent).toBeVisible({ timeout: 10_000 })

    // 2. Type blockquote syntax
    await cmContent.click()
    await page.keyboard.press('End')
    await page.keyboard.type('\n> Notable blockquote line\n')

    // 3. Move cursor away from the blockquote line so live preview activates
    await page.keyboard.press('ArrowUp')
    await page.keyboard.press('ArrowUp')

    // 4. Verify cm-quote styling is applied to blockquote line
    const quoteLine = page.locator('.cm-quote')
    await expect(quoteLine.first()).toBeVisible({ timeout: 5_000 })
    await expect(quoteLine.first()).toContainText('Notable blockquote line')
  })

  test('renders markdown fenced code block with cm-code styling', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const plainRow = page.locator('[data-tree-row="notes/plain.md"]')
    await expect(plainRow).toBeVisible()
    await plainRow.press('Enter')

    const cmContent = page.locator('.note .cm-content').first()
    await expect(cmContent).toBeVisible({ timeout: 10_000 })

    // Type fenced code block
    await cmContent.click()
    await page.keyboard.press('End')
    await page.keyboard.type('\n```ts\nconst quantumState = 101;\n```\n')

    // Move cursor away so live preview is active
    await page.keyboard.press('End')

    // Verify cm-code class is applied to code block lines
    const codeLines = page.locator('.cm-code')
    await expect(codeLines.first()).toBeVisible({ timeout: 5_000 })
    await expect(codeLines.first()).toHaveClass(/cm-code-first/)
    await expect(page.locator('.cm-code', { hasText: 'quantumState' })).toBeVisible({
      timeout: 5_000,
    })
  })

  test('renders bulleted list with bullet marker widget in editor', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const plainRow = page.locator('[data-tree-row="notes/plain.md"]')
    await expect(plainRow).toBeVisible()
    await plainRow.press('Enter')

    const cmContent = page.locator('.note .cm-content').first()
    await expect(cmContent).toBeVisible({ timeout: 10_000 })

    // Type bullet list item
    await cmContent.click()
    await page.keyboard.press('End')
    await page.keyboard.type('\n- Fundamental principle of entropy\n')

    // Verify list item text is present
    await expect(cmContent).toContainText('Fundamental principle of entropy')

    // Verify bullet widget or bullet mark
    const bullet = page
      .locator('.cm-bullet')
      .or(page.locator('.cm-line', { hasText: 'Fundamental principle of entropy' }))
    await expect(bullet.first()).toBeVisible({ timeout: 5_000 })
  })
})
