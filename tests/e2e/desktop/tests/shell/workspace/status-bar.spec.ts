import { test, expect } from '../../support/test.js'

test.describe('Status Bar and Task Popover Interactions', () => {
  test('opens status bar background task popover on trigger click', async ({ page }) => {
    await page.goto('/')

    const trigger = page.locator('.status-bar__trigger')
    await expect(trigger).toBeVisible()

    // Click trigger to open background task popover
    await trigger.click()

    await expect(trigger).toHaveAttribute('aria-expanded', 'true')
    const popover = page.locator('.task-popover')
    await expect(popover).toBeVisible()
    await expect(popover.locator('.task-popover__title')).toHaveText(/Background tasks/i)

    // The popover lists the tasks the core reports, and says so when it reports none.
    await expect
      .poll(async () => {
        const tasks = await popover.locator('.task-popover__body').count()
        const empty = await popover.locator('.task-popover__empty').count()
        return tasks + empty
      })
      .toBe(1)
  })

  test('dismisses status bar popover via escape key', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('.status-bar__trigger')).toBeVisible()

    const trigger = page.locator('.status-bar__trigger')
    await trigger.click()

    const popover = page.locator('.task-popover')
    await expect(popover).toBeVisible()
    await expect(trigger).toHaveAttribute('aria-expanded', 'true')

    // Press Escape to dismiss
    await page.keyboard.press('Escape')

    await expect(popover).toBeHidden()
    await expect(trigger).toHaveAttribute('aria-expanded', 'false')
  })

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
