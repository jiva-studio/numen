import { test, expect, type Page } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'

const openLinkInspector = async (page: Page) => {
  await openWorkspace(page)
  await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

  const edgeHit = getGraph(page).locator('.plex__edge-hit').first()
  await expect(edgeHit).toBeAttached()
  await edgeHit.dispatchEvent('click', { clientX: 300, clientY: 300 })

  const popover = page.locator('.plex-link-popover')
  await expect(popover).toBeVisible()
  return popover
}

test.describe('Plex Edge and Link Inspector', () => {
  test('opens link inspector popover on plex edge click', async ({ page }) => {
    const popover = await openLinkInspector(page)

    await expect(popover.locator('.plex-link-popover__title')).toBeVisible()
    await expect(popover.getByRole('checkbox', { name: 'Thermodynamics Entropy' })).toBeChecked()
    await expect(
      popover.getByRole('checkbox', { name: 'Entropy Thermodynamics' }),
    ).not.toBeChecked()
  })

  test('toggles the arrow of a link inside link inspector', async ({ page }) => {
    const popover = await openLinkInspector(page)

    const arrow = popover.getByRole('button', { name: 'Arrow' }).first()
    await expect(arrow).toHaveAttribute('aria-pressed', 'true')

    await arrow.click()

    await expect(arrow).toHaveAttribute('aria-pressed', 'false')
  })

  test('dismisses link inspector via close button without breaking plex graph', async ({
    page,
  }) => {
    const popover = await openLinkInspector(page)

    const closeBtn = popover.locator('.plex-link-popover__close-btn')
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()

    await expect(popover).not.toBeVisible()

    const thermoNode = page.locator('.plex__title', { hasText: 'Thermodynamics' })
    await expect(thermoNode).toBeVisible()
  })
})
