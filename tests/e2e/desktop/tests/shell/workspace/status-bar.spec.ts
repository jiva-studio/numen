import { test, expect } from '../../support/test.js'

test.describe('Status Bar and Task Popover Interactions', () => {

  test('dismisses status bar popover via explicit close button', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('.status-bar__trigger')).toBeVisible()

    const trigger = page.locator('.status-bar__trigger')
    await trigger.click()

    const popover = page.locator('.task-popover')
    await expect(popover).toBeVisible()

    // Click the popover close button
    const closeBtn = page.locator('.task-popover__close-btn')
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()

    await expect(popover).toBeHidden()
    await expect(trigger).toHaveAttribute('aria-expanded', 'false')
  })
})
