import { test, expect } from '../../support/test.js'
import { openVaultFile, openWorkspace } from '../../support/workspace.js'
import {
  createCardId,
  createDeckText,
  deleteVaultFiles,
  waitForListedFile,
} from '../../support/media.js'
import { createUniqueTitle, readVaultFile, writeVaultFile } from '../../support/vault.js'

test.describe('the schedule line of a deck', () => {
  let preset: string
  let deck: string

  test.beforeEach(async ({ request }) => {
    preset = createUniqueTitle('Estuary pace')
    deck = createUniqueTitle('Estuary birds')
    await writeVaultFile(
      request,
      `${preset}.md`,
      `---\ntype: preset\ngoal: retention\nretention: 0.9\n---\n\n# ${preset}\n`,
    )
    await writeVaultFile(
      request,
      `cards/${deck}.md`,
      createDeckText(deck, [
        { id: createCardId(), cardModel: 'Animal', values: { Name: 'Avocet', Height: '45 cm' } },
      ]),
    )
    await waitForListedFile(request, `${preset}.md`)
    await waitForListedFile(request, `cards/${deck}.md`)
  })

  test.afterEach(() => {
    deleteVaultFiles(`${preset}.md`, `cards/${deck}.md`)
  })

  test('writes the preset chosen from its menu into the deck file, and Defaults takes it out', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    let panel = await openVaultFile(page, 'cards', `${deck}.md`)
    const choice = panel.locator('.deck-tab__choice')
    await expect(choice).toHaveText('Defaults')

    await choice.click()
    await page.getByRole('menuitemradio', { name: preset }).click()

    await expect(choice).toHaveText(preset)
    await expect
      .poll(() => readVaultFile(request, `cards/${deck}.md`))
      .toMatch(new RegExp(`- to: ${preset}\\n\\s+role: ref\\n\\s+type: preset`))
    expect(await readVaultFile(request, `cards/${deck}.md`)).toContain('## Avocet ^')

    await page.reload()
    await openWorkspace(page)
    panel = await openVaultFile(page, 'cards', `${deck}.md`)
    await expect(panel.locator('.deck-tab__choice')).toHaveText(preset)

    await panel.locator('.deck-tab__choice').click()
    await page.getByRole('menuitemradio', { name: 'Defaults' }).click()

    await expect(panel.locator('.deck-tab__choice')).toHaveText('Defaults')
    await expect
      .poll(() => readVaultFile(request, `cards/${deck}.md`))
      .not.toContain('type: preset')
  })

  test('closes its menu on Escape and leaves the deck file as it was', async ({
    page,
    request,
  }) => {
    const before = await readVaultFile(request, `cards/${deck}.md`)
    await openWorkspace(page)
    const panel = await openVaultFile(page, 'cards', `${deck}.md`)

    await panel.locator('.deck-tab__choice').click()
    const menu = page.getByRole('menu', { name: 'Scheduled by' })
    await expect(menu.getByRole('menuitemradio', { name: preset })).toBeVisible()
    await page.keyboard.press('Escape')

    await expect(menu).toHaveCount(0)
    await expect(panel.locator('.deck-tab__choice')).toHaveText('Defaults')
    expect(await readVaultFile(request, `cards/${deck}.md`)).toBe(before)
  })
})
