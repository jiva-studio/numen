import { test, expect } from 'playwright/test'

test.describe('Plex Edge and Link Inspector', () => {
  test('opens link inspector popover on plex edge click', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('.plex')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached({ timeout: 10_000 })

    const edgeHit = page.locator('.plex__edge-hit').first()
    await expect(edgeHit).toBeAttached({ timeout: 10_000 })

    // Dispatch click on edge hit path
    await edgeHit.dispatchEvent('click', { clientX: 300, clientY: 300 })

    const popover = page.locator('.plex-link-popover')
    await expect(popover).toBeVisible({ timeout: 10_000 })
    await expect(popover.locator('.plex-link-popover__title')).toBeVisible()
    await expect(popover.locator('.plex-link-row')).toBeVisible()
  })

  test('toggles link directionality inside link inspector', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('.plex')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached({ timeout: 10_000 })

    const edgeHit = page.locator('.plex__edge-hit').first()
    await expect(edgeHit).toBeAttached({ timeout: 10_000 })
    await edgeHit.dispatchEvent('click', { clientX: 300, clientY: 300 })

    const popover = page.locator('.plex-link-popover')
    await expect(popover).toBeVisible({ timeout: 10_000 })

    const dirBtn = popover.locator('.plex-link-popover__dir-btn').first()
    await expect(dirBtn).toBeVisible()

    // Read initial HTML or icon state
    const initialSvg = await dirBtn.innerHTML()

    // Click to toggle direction
    await dirBtn.click()

    // Verify button SVG icon toggled
    const updatedSvg = await dirBtn.innerHTML()
    expect(updatedSvg).not.toBe(initialSvg)
  })

  test('dismisses link inspector via close button without breaking plex graph', async ({
    page,
  }) => {
    await page.goto('/')

    await expect(page.locator('.plex')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached({ timeout: 10_000 })

    const edgeHit = page.locator('.plex__edge-hit').first()
    await expect(edgeHit).toBeAttached({ timeout: 10_000 })
    await edgeHit.dispatchEvent('click', { clientX: 300, clientY: 300 })

    const popover = page.locator('.plex-link-popover')
    await expect(popover).toBeVisible({ timeout: 10_000 })

    // Click close button
    const closeBtn = popover.locator('.plex-link-popover__close-btn')
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()

    await expect(popover).not.toBeVisible()

    // Verify Plex graph nodes remain intact and visible
    const thermoNode = page.locator('.plex__title', { hasText: 'Thermodynamics' })
    await expect(thermoNode).toBeVisible()
  })
})
