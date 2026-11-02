import { test, expect, type Page } from '../../support/test.js'
import {
  openTaskList,
  openVaultFile,
  openWorkspace,
  runPaletteCommand,
} from '../../support/workspace.js'
import {
  createCardId,
  createDeckText,
  deleteVaultFiles,
  waitForListedFile,
} from '../../support/media.js'
import { createUniqueTitle, readVaultFile, writeVaultFile } from '../../support/vault.js'

/** Runs a palette command that asks for a name, and gives it the name. */
const runNamingCommand = async (page: Page, command: string, name: string): Promise<void> => {
  await runPaletteCommand(page, command, command)
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(name)
  await page.keyboard.press('Enter')
}

/** The number a frontmatter key of a preset file holds. */
const readNumber = (text: string, key: string): number =>
  Number(new RegExp(`^${key}: (.+)$`, 'm').exec(text)?.[1] ?? Number.NaN)

test.describe('the preset editor', () => {
  const made: string[] = []

  test.afterEach(() => {
    deleteVaultFiles(...made.splice(0))
  })

  test('New preset writes a preset file and opens its goal, minutes per day first', async ({
    page,
    request,
  }) => {
    const title = createUniqueTitle('Tidepool pace')
    made.push(`${title}.md`)
    await openWorkspace(page)

    await runNamingCommand(page, 'New preset', title)

    await expect(page.getByRole('tab', { name: new RegExp(title) })).toBeVisible()
    await expect(page.getByRole('radio', { name: 'Minutes per day' })).toBeChecked()
    await expect(page.getByRole('toolbar', { name: 'Load by weekday' })).toBeVisible()
    await expect.poll(() => readVaultFile(request, `${title}.md`)).toMatch(/^type: preset$/m)
  })

  test('writes the goal chosen to the preset file, and shows it chosen after a reload', async ({
    page,
    request,
  }) => {
    const title = createUniqueTitle('Tidepool goal')
    made.push(`${title}.md`)
    await openWorkspace(page)
    await runNamingCommand(page, 'New preset', title)
    await expect(page.getByRole('radio', { name: 'Minutes per day' })).toBeChecked()

    await page.getByRole('radio', { name: 'Retention' }).click()

    await expect.poll(() => readVaultFile(request, `${title}.md`)).toMatch(/^goal: retention$/m)

    await page.reload()
    await openWorkspace(page)
    await page.getByRole('tab', { name: /^Files/ }).click()
    await page.getByRole('treeitem', { name: `${title}.md`, exact: true }).press('Enter')
    await expect(page.getByRole('radio', { name: 'Retention' })).toBeChecked()
  })

  test('opens the preset a deck names, and writes the retention its slider is moved to', async ({
    page,
    request,
  }) => {
    const preset = createUniqueTitle('Tidepool recall')
    const deck = createUniqueTitle('Tidepool animals')
    made.push(`${preset}.md`, `cards/${deck}.md`)
    await writeVaultFile(
      request,
      `${preset}.md`,
      `---\ntype: preset\ngoal: retention\nretention: 0.9\n---\n\n# ${preset}\n`,
    )
    const deckText = createDeckText(deck, [
      { id: createCardId(), cardModel: 'Animal', values: { Name: 'Limpet', Height: '2 cm' } },
      { id: createCardId(), cardModel: 'Animal', values: { Name: 'Whelk', Height: '5 cm' } },
    ]).replace(
      'type: deck\n',
      `type: deck\nlinks:\n  - to: ${preset}\n    role: ref\n    type: preset\n`,
    )
    await writeVaultFile(request, `cards/${deck}.md`, deckText)
    await waitForListedFile(request, `${preset}.md`)
    await waitForListedFile(request, `cards/${deck}.md`)

    await openWorkspace(page)
    await openVaultFile(page, 'cards', `${deck}.md`)
    await runPaletteCommand(page, 'Open preset', 'Open preset tab')
    await expect(page.getByRole('tab', { name: new RegExp(preset) })).toBeVisible()
    const slider = page.getByRole('slider', { name: 'Preset goal' })
    await expect(slider).toBeVisible()
    const before = await slider.getAttribute('aria-valuetext')

    await slider.press('ArrowLeft')

    await expect(slider).not.toHaveAttribute('aria-valuetext', before ?? '')
    await expect
      .poll(async () => readNumber(await readVaultFile(request, `${preset}.md`), 'retention'))
      .not.toBe(0.9)
    const retention = readNumber(await readVaultFile(request, `${preset}.md`), 'retention')
    expect(retention).toBeGreaterThan(0.5)
    expect(retention).toBeLessThan(0.9)
    const moved = await slider.getAttribute('aria-valuetext')

    await page.reload()
    await openWorkspace(page)
    await page.getByRole('tab', { name: /^Files/ }).click()
    await page.getByRole('treeitem', { name: `${preset}.md`, exact: true }).press('Enter')
    await expect(page.getByRole('slider', { name: 'Preset goal' })).toHaveAttribute(
      'aria-valuetext',
      moved ?? '',
    )
  })

  test('says a deck naming no preset is scheduled by the defaults', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'cards', 'Animals.md')

    await runPaletteCommand(page, 'Open preset', 'Open preset tab')

    await expect(await openTaskList(page)).toContainText(
      'No preset set. Scheduled by the defaults.',
    )
    await expect(page.getByRole('region', { name: 'Goal', exact: true })).toHaveCount(0)
  })
})
