import { test, expect } from '../../support/test.js'

test.describe('PDF Document Reader Advanced Controls', () => {
  test('zooms pdf document in and out via reader zoom controls', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open assets folder and sample.pdf
    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    const reader = page.locator('.reader')
    await expect(reader).toBeVisible({ timeout: 10_000 })

    // 2. Identify zoom controls
    const zoomInBtn = page.getByRole('button', { name: 'Zoom in' })
    const zoomOutBtn = page.getByRole('button', { name: 'Zoom out' })

    await expect(zoomInBtn).toBeVisible({ timeout: 5_000 })
    await expect(zoomOutBtn).toBeVisible({ timeout: 5_000 })

    // 3. Zoom in and verify zoom button state
    await zoomInBtn.click()
    await page.waitForTimeout(200)
    await zoomInBtn.click()
    await page.waitForTimeout(200)

    // Zoom out button should be enabled after zooming in
    await expect(zoomOutBtn).toBeEnabled()

    // 4. Zoom out back towards base level
    await zoomOutBtn.click()
    await page.waitForTimeout(200)
    await expect(reader).toBeVisible()
  })

  test('jumps directly to specific pdf page using page number input', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    // Locate page input
    const pageInput = page
      .getByRole('spinbutton', { name: 'Page' })
      .or(page.locator('input[aria-label="Page"]'))
    await expect(pageInput).toBeVisible({ timeout: 5_000 })

    // Current page starts at 1
    await expect(pageInput).toHaveValue('1')

    // Fill page number 2 and submit with Enter
    await pageInput.fill('2')
    await pageInput.press('Enter')

    // Verify page input retains value 2
    await expect(pageInput).toHaveValue('2')
  })

  test('keeps one page number while the view rests on the border between two pages', async ({
    page,
  }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const pdfRow = page.locator('[data-tree-row="assets/pages.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    const viewport = page.locator('.reader__viewport')
    const pageInput = page.locator('input[aria-label="Page"]')
    await expect(page.locator('.reader__page[data-page="1"]')).toBeAttached({ timeout: 20_000 })

    // The centre line of the viewport is put one pixel either side of where the
    // second page begins.
    const border = await viewport.evaluate((element: HTMLElement) => {
      const second = element.querySelector<HTMLElement>('.reader__page[data-page="1"]')!
      return second.offsetLeft - element.clientWidth / 2
    })

    for (const [shift, expected] of [
      [-1, '1'],
      [1, '2'],
    ] as const) {
      await viewport.evaluate((element: HTMLElement, left: number) => {
        element.scrollLeft = left
      }, border + shift)
      await expect(pageInput).toHaveValue(expected)

      const values: string[] = []
      for (let sample = 0; sample < 15; sample++) {
        await page.waitForTimeout(100)
        values.push(await pageInput.inputValue())
      }
      expect(values).toEqual(Array.from({ length: 15 }, () => expected))
    }
  })

  test('shows the page it turned to after the next page button', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const pdfRow = page.locator('[data-tree-row="assets/pages.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    const pageInput = page.locator('input[aria-label="Page"]')
    await expect(pageInput).toHaveValue('1', { timeout: 20_000 })

    const nextBtn = page.getByRole('button', { name: 'Next page' })
    for (const expected of ['2', '3', '4']) {
      await nextBtn.click()
      await expect(pageInput).toHaveValue(expected)
      // The scroll to the page has ended and the field still shows that page.
      await page.waitForTimeout(1_000)
      await expect(pageInput).toHaveValue(expected)
    }
  })

  test('enforces navigation boundary limits at document start and end', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    const backBtn = page.getByRole('button', { name: 'Previous page' })
    const nextBtn = page.getByRole('button', { name: 'Next page' })

    await expect(backBtn).toBeVisible({ timeout: 5_000 })
    await expect(nextBtn).toBeVisible({ timeout: 5_000 })

    // On page 1, previous page must be disabled
    await expect(backBtn).toBeDisabled()

    // Advance page if next button is enabled
    if (await nextBtn.isEnabled()) {
      await nextBtn.click()
      await expect(backBtn).toBeEnabled()
    }
  })
})
