import { test, expect, readSavedStates } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  getNoteTab,
  getTabName,
  getNoteText,
  openNote,
  readNote,
  typeAtEnd,
  waitForSavedBody,
  writeNote,
} from '../../support/vault.js'

test.describe('Note editor', () => {
  test('saves a typed note to disk without Ctrl+S, and shows it after a reload', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Brine kettle'))
    try {
      await writeNote(request, notePath, 'The kettle boils seawater for salt.\n')
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text).toContainText('The kettle boils seawater for salt.')

      await typeAtEnd(page, text, '\nThe pans dry in the wind.')

      await waitForSavedBody(request, notePath, 'The pans dry in the wind.')
      expect((await readNote(request, notePath)).body).toContain(
        'The kettle boils seawater for salt.',
      )
      await expect.poll(async () => (await readSavedStates(request)).has(notePath)).toBe(true)

      await page.reload()

      await expect(getNoteText(page, notePath)).toContainText('The pans dry in the wind.')
    } finally {
      await destroyNote(request, notePath)
    }
  })

  test('writes what was typed to disk when the tab is closed at once', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Salt pan'))
    try {
      await writeNote(request, notePath, 'The pan is raked at noon.\n')
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text).toContainText('The pan is raked at noon.')

      await typeAtEnd(page, text, ' The crust is lifted at dusk.')
      await page.getByRole('button', { name: `Close ${getTabName(notePath)}`, exact: true }).click()
      await expect(getNoteTab(page, notePath)).toHaveCount(0)

      await waitForSavedBody(request, notePath, 'The crust is lifted at dusk.')
    } finally {
      await destroyNote(request, notePath)
    }
  })
})
