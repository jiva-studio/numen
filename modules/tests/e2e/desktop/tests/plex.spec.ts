import { test, expect } from 'playwright/test'

test.describe('Plex Graph Interaction', () => {
  test('interacts with knowledge graph nodes', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    const thermoNode = page.getByText('Thermodynamics').first()
    const entropyNode = page.getByText('Entropy').first()

    await expect(thermoNode).toBeVisible()
    await expect(entropyNode).toBeVisible()

    // Click on Entropy node to refocus the graph neighbourhood
    await entropyNode.click()

    // After clicking Entropy, it remains visible in the neighbourhood
    await expect(entropyNode).toBeVisible()
  })
})
