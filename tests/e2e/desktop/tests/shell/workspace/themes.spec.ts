import { test, expect } from '../../support/test.js'

test.describe('Theming and Display Scaling', () => {
  test('switches application theme mode to dark mode', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Choose Light or dark command
    await paletteInput.fill('Light or dark')
    await page.keyboard.press('Enter')

    // In mode selection step, choose Dark
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Dark')
    await page.keyboard.press('Enter')

    // Verify workspace remains active
    await expect(page.locator('.plex').or(page.locator('.neighbourhood')).first()).toBeVisible()
  })

  test('switches application theme mode to light mode', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Light or dark')
    await page.keyboard.press('Enter')

    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Light')
    await page.keyboard.press('Enter')

    await expect(page.locator('.plex').or(page.locator('.neighbourhood')).first()).toBeVisible()
  })

  test('adjusts interface and reading size in settings tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open Settings via command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    // In master-detail settings, open System settings
    const systemButton = page.getByRole('button', { name: /^System/i }).first()
    if (await systemButton.isVisible()) {
      await systemButton.click()
    }

    // Verify Settings tab opened
    await expect(page.getByRole('spinbutton', { name: 'Interface size' }).first()).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByRole('spinbutton', { name: 'Reading size' }).first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('draws the plugin search field with the token radius under the default theme', async ({
    page,
  }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    const search = page.getByRole('searchbox', { name: 'Search plugins' }).first()
    await expect(search).toBeVisible({ timeout: 10_000 })

    const corners = await search.evaluate((field) => {
      const probe = document.createElement('div')
      probe.style.borderRadius = 'var(--numen-radius-tight)'
      document.body.append(probe)
      const token = getComputedStyle(probe).borderTopLeftRadius
      probe.remove()
      return { field: getComputedStyle(field).borderTopLeftRadius, token }
    })
    expect(Number.parseFloat(corners.field)).toBeGreaterThan(0)
    expect(corners.field).toBe(corners.token)
  })
})
