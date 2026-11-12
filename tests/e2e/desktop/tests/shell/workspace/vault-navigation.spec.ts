import { test, expect, readActivePath } from '../../support/test.js'
import {
  getGraph,
  getPaletteField,
  openWorkspace,
  runPaletteCommand,
} from '../../support/workspace.js'

test.describe('Vault Navigation and Quick Open Palette', () => {
  test('shows the note picked in Go to note in the plex with the command Show in plex', async ({
    page,
  }) => {
    await openWorkspace(page)
    await expect(getGraph(page).getByRole('img', { name: 'Entropy, focus' })).toBeHidden()

    await runPaletteCommand(page, 'Go to note', 'Go to note')
    await getPaletteField(page).fill('Entropy')
    await expect(page.getByRole('option', { selected: true })).toContainText('Entropy')
    await page.keyboard.press('Enter')
    await expect.poll(() => readActivePath(page.request)).toMatch(/Entropy\.md$/)

    await runPaletteCommand(page, 'Show in plex', 'Show in plex')

    await expect(getGraph(page).getByRole('img', { name: 'Entropy, focus' })).toBeAttached()
    await expect(
      getGraph(page).getByRole('button', { name: 'Thermodynamics, parent' }),
    ).toBeAttached()
  })
})
