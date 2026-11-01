import { test, expect, readSavedStates } from '../../support/test.js'
import { openWorkspace, openVaultFile } from '../../support/workspace.js'
import { readSettledWidth, readWidth } from '../../support/layout.js'
import { observeFor, waitForFrames } from '../../support/browser.js'

test.describe('PDF Document Reader Advanced Controls', () => {
  test('draws the pages wider with each press of Zoom in and narrower with Zoom out', async ({
    page,
  }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'pages.pdf')
    const firstPage = page.locator('.reader__page[data-page="0"]')
    await expect(firstPage).toBeAttached({ timeout: 20_000 })
    const base = await readSettledWidth(firstPage)

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => readWidth(firstPage)).toBeCloseTo(base * 1.25, -1)
    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => readWidth(firstPage)).toBeCloseTo(base * 1.25 * 1.25, -1)

    await page.getByRole('button', { name: 'Zoom out' }).click()
    await expect.poll(() => readWidth(firstPage)).toBeCloseTo(base * 1.25, -1)
    await page.getByRole('button', { name: 'Zoom out' }).click()
    await expect.poll(() => readWidth(firstPage)).toBeCloseTo(base, -1)
    await expect(page.getByRole('button', { name: 'Zoom out' })).toBeDisabled()
  })

  test('scrolls to the page typed in the page field and keeps it in view', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'pages.pdf')
    const viewport = page.locator('.reader__viewport')
    const numberField = page.getByLabel('Page', { exact: true })
    await expect(numberField).toHaveValue('1')

    await numberField.fill('12')
    await numberField.press('Enter')

    await expect(numberField).toHaveValue('12')
    const target = page.locator('.reader__page[data-page="11"]')
    await expect(target).toBeAttached({ timeout: 20_000 })
    await expect
      .poll(async () => {
        const box = await target.boundingBox()
        const frame = await viewport.boundingBox()
        if (!box || !frame) return false
        const centre = frame.x + frame.width / 2
        return box.x <= centre && centre <= box.x + box.width
      })
      .toBe(true)
    await expect
      .poll(async () => (await readSavedStates(page.request)).get('assets/pages.pdf'))
      .toContain('"page":12')
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
    const numberField = page.getByLabel('Page', { exact: true })
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
      await expect(numberField).toHaveValue(expected)

      const values: string[] = []
      for (let sample = 0; sample < 15; sample++) {
        await waitForFrames(page, 6)
        values.push(await numberField.inputValue())
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

    const numberField = page.getByLabel('Page', { exact: true })
    await expect(numberField).toHaveValue('1', { timeout: 20_000 })

    const nextBtn = page.getByRole('button', { name: 'Next page' })
    for (const expected of ['2', '3', '4']) {
      await nextBtn.click()
      await expect(numberField).toHaveValue(expected)
      // The scroll to the page has ended and the field still shows that page.
      await observeFor(page, 1_000)
      await expect(numberField).toHaveValue(expected)
    }
  })

  test('enforces navigation boundary limits at document start and end', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'pages.pdf')
    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })
    const numberField = page.getByLabel('Page', { exact: true })
    await expect(numberField).toHaveValue('1', { timeout: 20_000 })

    const backBtn = page.getByRole('button', { name: 'Previous page' })
    const nextBtn = page.getByRole('button', { name: 'Next page' })

    // On page 1, previous page is disabled and next page is not.
    await expect(backBtn).toBeDisabled()
    await expect(nextBtn).toBeEnabled()
    await nextBtn.click()
    await expect(backBtn).toBeEnabled()

    // On the last page of the 40, next page is disabled and previous page is not.
    await numberField.fill('40')
    await numberField.press('Enter')
    await expect(numberField).toHaveValue('40')
    await expect(nextBtn).toBeDisabled()
    await expect(backBtn).toBeEnabled()
  })
})
