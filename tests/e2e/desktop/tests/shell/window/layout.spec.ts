import { test, expect, type Locator, type Page } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import { readBox } from '../../support/browser.js'
import { getRow, getTab, readOpenTabs, readWidth, showInFiles } from '../../support/layout.js'

/** The panes of the window, in the order they are drawn. */
const getPanes = (page: Page): Locator => page.locator('section[data-workspace-pane]')

/** The pane that draws the tab of that title. */
const getPaneOf = (page: Page, title: string): Locator =>
  getPanes(page).filter({ has: getTab(page, title) })

test.describe('Workspace Layout and Split Panes', () => {
  test('opens the note in front in a pane of its own beside the others with Open beside', async ({
    page,
  }) => {
    await openWorkspace(page)
    await expect(getPanes(page)).toHaveCount(2)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    await expect(getTab(page, 'Entropy')).toBeVisible()

    // Open beside is the command Shift and Enter reach on the row of Open the note.
    await page.keyboard.press('Control+p')
    await page.locator('input[data-palette="field"]').fill('Open the note')
    await expect(page.getByRole('option', { name: /^Open the note/ })).toBeVisible()
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

  test('moves the line between two panes by the distance the separator is dragged', async ({
    page,
  }) => {
    await openWorkspace(page)
    const separator = page.getByRole('separator', { name: 'Resize panes' })
    const main = getPanes(page).nth(0)
    const aside = getPanes(page).nth(1)
    const mainBefore = await readWidth(main)
    const asideBefore = await readWidth(aside)
    const box = await readBox(separator)

    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
    await page.mouse.down()
    await page.mouse.move(box.x + box.width / 2 - 150, box.y + box.height / 2, { steps: 10 })
    await page.mouse.up()

    await expect.poll(async () => readWidth(main)).toBeLessThan(mainBefore - 140)
    expect(await readWidth(main)).toBeGreaterThan(mainBefore - 160)
    expect(await readWidth(aside)).toBeGreaterThan(asideBefore + 140)
    expect(await readWidth(aside)).toBeLessThan(asideBefore + 160)
    await expect(getGraph(page)).toBeVisible()
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
    expect((await readOpenTabs(request)).tabs.map((one) => one.path).sort()).toEqual(before)
  })
})
