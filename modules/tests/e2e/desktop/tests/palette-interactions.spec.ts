import { test, expect } from 'playwright/test'

test.describe('Command Palette Advanced Keyboard and Pointer Interactions', () => {
  test('navigates command palette options using keyboard arrow keys', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Verify selectable item rows are present
    const items = page.locator('.palette__item')
    await expect(items.first()).toBeVisible({ timeout: 5_000 })

    // Find the currently selected item
    const selectedItem = page.locator('.palette__item[aria-selected="true"]').first()
    await expect(selectedItem).toBeVisible({ timeout: 5_000 })
    const initialText = await selectedItem.innerText()

    // 2. Press ArrowDown to advance selection
    await page.keyboard.press('ArrowDown')

    // Verify selection moved to another item
    const nextSelectedItem = page.locator('.palette__item[aria-selected="true"]').first()
    await expect(nextSelectedItem).toBeVisible({ timeout: 5_000 })
    const nextText = await nextSelectedItem.innerText()
    expect(nextText).not.toBe(initialText)

    // 3. Press ArrowUp to return to initial item
    await page.keyboard.press('ArrowUp')
    const restoredSelectedItem = page.locator('.palette__item[aria-selected="true"]').first()
    await expect(restoredSelectedItem).toBeVisible({ timeout: 5_000 })
    expect(await restoredSelectedItem.innerText()).toBe(initialText)
  })

  test('displays polite empty state on query with no matching commands', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Type completely unmatched query
    await paletteInput.fill('xyzunmatchedquery99999')
    await page.waitForTimeout(300)

    // Verify no selectable item rows remain
    const items = page.locator('.palette__item')
    await expect(items).toHaveCount(0)

    // Verify polite aria announcement element
    const said = page.locator('[data-palette="said"]')
    await expect(said).toBeAttached()
  })

  test('navigates multi-step palette flows and cancels step on escape', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Choose New note command
    await paletteInput.fill('New note')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Second step expects the note name input
    await page.waitForTimeout(300)
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // First Escape leaves the child step and returns to root palette
    await page.keyboard.press('Escape')
    const panel = page.locator('[data-palette="panel"]')
    await expect(panel).toBeVisible({ timeout: 5_000 })

    // Second Escape closes palette completely
    await page.keyboard.press('Escape')
    await expect(panel).not.toBeVisible({ timeout: 5_000 })
  })

  test('dismisses command palette on background scrim click', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const panel = page.locator('[data-palette="panel"]').first()
    await expect(panel).toBeVisible({ timeout: 5_000 })

    // Click on the scrim (ground) outside the palette panel
    const ground = page.locator('[data-palette="ground"]').first()
    await ground.click({ position: { x: 10, y: 10 } })

    // Verify palette panel is dismissed
    await expect(panel).not.toBeVisible({ timeout: 5_000 })
  })
})
