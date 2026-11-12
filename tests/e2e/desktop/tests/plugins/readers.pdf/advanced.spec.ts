import { test, expect, readSavedStates } from '../../support/test.js'
import { openAsset, openVaultFile, openWorkspace } from '../../support/workspace.js'
import { readSettledWidth } from '../../support/layout.js'
import { observeFor } from '../../support/browser.js'

test.describe('PDF Document Reader Advanced Controls', () => {
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

  test('shows the page that holds the centre line of the view on either side of a border', async ({
    page,
  }) => {
    await openAsset(page, 'pages.pdf')

    const viewport = page.locator('.reader__viewport')
    const numberField = page.getByLabel('Page', { exact: true })
    const second = page.locator('.reader__page[data-page="1"]')
    await expect(second).toBeAttached({ timeout: 20_000 })
    // The row is laid out again while the viewport is measured and the page sizes arrive, and a
    // place measured before then is not the border any more. The first picture is drawn once the
    // width has settled.
    await expect(page.locator('.reader__page[data-page="0"] .reader__picture')).toBeVisible({
      timeout: 20_000,
    })
    await readSettledWidth(second)

    for (const [shift, expected] of [
      [-1, '1'],
      [1, '2'],
    ] as const) {
      // The centre line of the viewport is put one pixel either side of where the second page
      // begins, measured and scrolled to in one step so no layout change comes between.
      await viewport.evaluate((element: HTMLElement, offset: number) => {
        const next = element.querySelector<HTMLElement>('.reader__page[data-page="1"]')!
        element.scrollLeft = next.offsetLeft - element.clientWidth / 2 + offset
      }, shift)
      await expect(numberField).toHaveValue(expected)
    }
  })

  test('shows the page it turned to after the next page button', async ({ page }) => {
    await openAsset(page, 'pages.pdf')

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
