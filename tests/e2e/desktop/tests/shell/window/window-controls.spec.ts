import { test, expect, type Page } from '../../support/test.js'
import { getGraph, openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import { getRow, getTab, readComputedFontSize, readOpenTabs, showInFiles } from '../../support/layout.js'
import { readSettings, writeSetting } from '../../support/vault.js'

const PLEX_ID = 'md.numen.views.plex.graph'

/** The editor of the open note of that title. */
const getEditor = (page: Page, title: string) =>
  page.getByRole('tabpanel', { name: new RegExp(title) }).locator('.cm-editor')

/** Picks a value in the list a palette command opens on, by the start of its name. */
const chooseValue = async (page: Page, value: string): Promise<void> => {
  await page.getByRole('option', { name: new RegExp(`^${value}`) }).click()
  await expect(page.getByRole('dialog', { name: 'Search the vault' })).toHaveCount(0)
}

test.describe('Window Controls and Display Preferences', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['appearance', 'text_scale'], 1)
    await writeSetting(request, ['plugins', PLEX_ID, 'nest_headings'], true)
  })

  test('closes the active tab with Close tab', async ({ page, request }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    await expect(getTab(page, 'Entropy')).toHaveAttribute('aria-selected', 'true')
    await expect
      .poll(async () => (await readOpenTabs(request)).tabs.map((one) => one.path))
      .toContain('notes/Entropy.md')

    await runPaletteCommand(page, 'Close tab', 'Close tab')

    await expect(getTab(page, 'Entropy')).toHaveCount(0)
    await expect
      .poll(async () => (await readOpenTabs(request)).tabs.map((one) => one.path))
      .not.toContain('notes/Entropy.md')
  })

  test('draws note text at the reading font size chosen in the palette, also after a reload', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    const editor = getEditor(page, 'Entropy')
    await expect(editor).toBeVisible()
    const before = await readComputedFontSize(editor)

    await runPaletteCommand(page, 'Reading font size', 'Reading font size')
    await chooseValue(page, '150%')

    await expect.poll(async () => readComputedFontSize(editor)).toBeCloseTo(before * 1.5, 0)
    await expect
      .poll(async () => (await readSettings(request))['appearance'])
      .toMatchObject({ text_scale: 1.5 })

    await page.reload()

    const reloaded = getEditor(page, 'Entropy')
    await expect(reloaded).toBeVisible()
    await expect.poll(async () => readComputedFontSize(reloaded)).toBeCloseTo(before * 1.5, 0)
  })

  test('shows or hides the headings under the notes of the plex with the Nest headings setting', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    const headings = getGraph(page).locator('.plex__heading-text')
    const focusNode = getGraph(page).getByRole('img', { name: 'Thermodynamics, focus' })
    await focusNode.hover()
    await expect(headings.filter({ hasText: 'First law' })).toBeVisible()

    await runPaletteCommand(page, 'Nest headings', 'Plex Graph: Nest headings under notes')
    await chooseValue(page, 'Off')
    await expect
      .poll(async () => (await readSettings(request))['plugins'])
      .toMatchObject({ [PLEX_ID]: { nest_headings: false } })

    await page.mouse.move(0, 0)
    await focusNode.hover()
    // Longer than a hovered node takes to open its headings while they are on.
    await page.waitForTimeout(1_500)
    await expect(headings).toHaveCount(0)

    await runPaletteCommand(page, 'Nest headings', 'Plex Graph: Nest headings under notes')
    await chooseValue(page, 'On')
    await page.mouse.move(0, 0)
    await focusNode.hover()
    await expect(headings.filter({ hasText: 'First law' })).toBeVisible()
  })
})
