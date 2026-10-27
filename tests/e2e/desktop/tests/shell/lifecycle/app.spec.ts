import { test, expect } from '../../support/test.js'

test.describe('Desktop Application Shell', () => {
  test('loads application and displays workspace with Plex graph and Agent panel', async ({
    page,
  }) => {
    await page.goto('/')

    // Title verification
    await expect(page).toHaveTitle('numen')

    // Status bar shows Ready
    await expect(page.locator('text=Ready')).toBeVisible()

    // Verify Plex graph rendered nodes from test vault
    await expect(page.locator('.plex__title', { hasText: 'Thermodynamics' })).toBeVisible()
    await expect(page.locator('.plex__title', { hasText: 'Entropy' })).toBeVisible()

    // Verify Agent panel rendered
    await expect(page.locator('text=Ask about the vault')).toBeVisible()
  })
})
