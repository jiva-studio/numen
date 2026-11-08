import { test, expect, type Locator, type Page } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import { getRow, getTab, readOpenTabs, showInFiles } from '../../support/layout.js'

const getPanes = (page: Page): Locator => page.locator('section[data-workspace-pane]')

const getPaneOf = (page: Page, title: string): Locator =>
  getPanes(page).filter({ has: getTab(page, title) })

test.describe('Workspace Layout and Split Panes', () => {
  test('opens the note in front in a pane of its own next to the others with Open Aside', async ({
    page,
  }) => {
    await openWorkspace(page)
    await expect(getPanes(page)).toHaveCount(2)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    await expect(getTab(page, 'Entropy')).toBeVisible()

    // Open note beside is the command Shift and Enter reach on the row of Open note.
    await page.keyboard.press('Control+p')
    await page.locator('input[data-palette="field"]').fill('Open note')
    await expect(page.getByRole('option', { name: /^Open note/ })).toBeVisible()
    await page.keyboard.press('Shift+Enter')

    await expect(getPanes(page)).toHaveCount(3)
    await expect(getPaneOf(page, 'Entropy')).toHaveCount(1)
    await expect(getPaneOf(page, 'Entropy')).not.toContainText('Files')
    // The panes draw side by side: the plex, the files and the note are all in view at once.
    await expect(getGraph(page)).toBeVisible()
    await expect(page.getByRole('tabpanel', { name: /Entropy/ })).toBeVisible()
    await expect(getTab(page, 'Files')).toBeVisible()
    const boxes = await getPanes(page).evaluateAll((panes) =>
      panes.map((one) => one.getBoundingClientRect()),
    )
    for (let index = 1; index < boxes.length; index++) {
      expect(boxes[index]!.x).toBeGreaterThanOrEqual(boxes[index - 1]!.x + boxes[index - 1]!.width)
    }
  })

  test('reopens the open tabs and the active one after a reload', async ({ page, request }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes', 'assets'])
    await getRow(page, 'assets/sample.pdf').press('Enter')
    await expect(getTab(page, 'sample.pdf')).toBeVisible()
    await getTab(page, 'Files').click()
    await getRow(page, 'notes/Entropy.md').press('Enter')
    await expect(getTab(page, 'Entropy')).toHaveAttribute('aria-selected', 'true')
    await expect
      .poll(async () => {
        const { tabs, activeId } = await readOpenTabs(request)
        return tabs.find((one) => one.id === activeId)?.path
      })
      .toBe('notes/Entropy.md')
    const before = (await readOpenTabs(request)).tabs.map((one) => one.path).sort()

    await page.reload()

    await expect(getTab(page, 'Entropy')).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('tabpanel', { name: /Entropy/ })).toBeVisible()
    await expect(getTab(page, 'sample.pdf')).toBeVisible()
    await expect(getTab(page, 'Files')).toBeVisible()
    await expect(getTab(page, 'Thermodynamics')).toBeVisible()
    // The window writes its tabs to the core after it draws them.
    await expect
      .poll(async () => (await readOpenTabs(request)).tabs.map((one) => one.path).sort())
      .toEqual(before)
  })
})
