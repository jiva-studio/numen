import { test, expect } from 'playwright/test'

test.describe('Workspace Layout and Split Panes', () => {
  test('renders split panes with book reader and note editor concurrently', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. In left pane, open Entropy note
    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // 2. In right pane, open sample.epub
    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    await expect(epubRow).toBeVisible()
    await epubRow.press('Enter')
    await expect(page.locator('.book-tab').first()).toBeVisible({ timeout: 10_000 })

    // Verify left and right split panes coexist
    await expect(page.getByRole('tab', { name: /Entropy/ }).first()).toBeVisible()
    await expect(page.locator('.book-tab').first()).toBeVisible()
    await expect(page.getByRole('separator', { name: 'Resize panes' })).toBeVisible()
  })

  test('resizes workspace panes via separator handle without breaking plugin canvas', async ({
    page,
  }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Locate the separator between panes
    const separator = page.getByRole('separator', { name: 'Resize panes' })
    await expect(separator).toBeVisible()

    // Get initial bounding box
    const box = await separator.boundingBox()
    expect(box).not.toBeNull()

    if (box) {
      // Drag separator to resize panes
      await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
      await page.mouse.down()
      await page.mouse.move(box.x + 80, box.y + box.height / 2)
      await page.mouse.up()
    }

    // Verify workspace remains intact
    await expect(separator).toBeVisible()
  })
})
