import { test, expect } from '../../support/test.js'
import {
  getGraph,
  getPaletteField,
  openWorkspace,
  runPaletteCommand,
} from '../../support/workspace.js'
import { openNote } from '../../support/vault.js'

test.describe('Vault Navigation and Quick Open Palette', () => {
  // #1504: shows the note in the plex again once the search-result menu exists.
  test.fixme('puts the start note back in the focus of the plex with Go to start note', async ({
    page,
  }) => {
    await openWorkspace(page)
    await openNote(page, 'notes/Entropy.md')
    await runPaletteCommand(page, 'Show note in plex', 'Show note in plex')
    await expect(getGraph(page).getByRole('img', { name: 'Entropy, focus' })).toBeAttached()

    await runPaletteCommand(page, 'Go to start note', 'Go to start note')

    await expect(getGraph(page).getByRole('img', { name: 'Thermodynamics, focus' })).toBeAttached()
    await expect(getGraph(page).getByRole('button', { name: 'Entropy, child' })).toBeAttached()
  })

  // #1504: shows the note in the plex again once the search-result menu exists.
  test.fixme('puts the note picked in Go to note in the focus of the plex', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Go to note', 'Go to note')
    await getPaletteField(page).fill('Entropy')
    await expect(page.getByRole('option', { selected: true })).toContainText('Entropy')
    await page.keyboard.press('Enter')

    await expect(getGraph(page).getByRole('img', { name: 'Entropy, focus' })).toBeAttached()
    await expect(
      getGraph(page).getByRole('button', { name: 'Thermodynamics, parent' }),
    ).toBeAttached()
  })

  test('reveals active note in file tree sidebar via reveal command', async ({ page }) => {
    await openWorkspace(page)
    await openNote(page, 'notes/Entropy.md')

    await page.getByRole('tab', { name: /^Agent/ }).click()
    await expect(page.getByRole('tab', { name: /^Files/ })).not.toHaveAttribute(
      'aria-selected',
      'true',
    )

    await runPaletteCommand(page, 'Show note in file tree', 'Show note in file tree')

    await expect(page.getByRole('tab', { name: /^Files/ })).toHaveAttribute('aria-selected', 'true')
    await expect(page.locator('[data-tree-row="notes/Entropy.md"]')).toBeVisible()
  })
})
