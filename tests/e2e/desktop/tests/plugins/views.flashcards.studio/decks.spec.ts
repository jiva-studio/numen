import { test, expect, type APIRequestContext, type Page } from '../../support/test.js'
import { openVaultFile, openWorkspace } from '../../support/workspace.js'
import {
  createCardId,
  createDeckText,
  deleteVaultFiles,
  waitForListedFile,
} from '../../support/media.js'
import { createUniqueTitle, readNote, writeVaultFile } from '../../support/vault.js'

const createDeck = async (request: APIRequestContext) => {
  const title = createUniqueTitle('Shore birds')
  const path = `cards/${title}.md`
  const heron = createCardId()
  const otter = createCardId()
  await writeVaultFile(
    request,
    path,
    createDeckText(title, [
      { id: heron, cardModel: 'Animal', values: { Name: 'Heron', Height: 'about 1 m' } },
      { id: otter, cardModel: 'Animal', values: { Name: 'Otter', Height: 'about 30 cm' } },
    ]),
  )
  await waitForListedFile(request, path)
  return { title, path, heron, otter }
}

const openDeck = async (page: Page, title: string) => {
  await openWorkspace(page)
  const panel = await openVaultFile(page, 'cards', `${title}.md`)
  await expect(panel.locator('.card').first()).toBeVisible()
  return panel
}

test.describe('the deck editor', () => {
  let deck: Awaited<ReturnType<typeof createDeck>>

  test.beforeEach(async ({ request }) => {
    deck = await createDeck(request)
  })

  test.afterEach(() => {
    deleteVaultFiles(deck.path)
  })

  test('writes an edited value to the deck file, and shows it after a reload', async ({
    page,
    request,
  }) => {
    let panel = await openDeck(page, deck.title)

    await panel
      .getByRole('article', { name: 'Card 1' })
      .getByRole('textbox', { name: 'Height' })
      .fill('about 1.2 m at the shoulder')

    await expect
      .poll(async () => (await readNote(request, deck.path)).body)
      .toContain('### Height\n\nabout 1.2 m at the shoulder')
    const body = (await readNote(request, deck.path)).body
    expect(body).not.toContain('about 1 m\n')
    expect(body).toContain('### Height\n\nabout 30 cm')

    await page.reload()
    panel = await openDeck(page, deck.title)
    await expect(
      panel.getByRole('article', { name: 'Card 1' }).getByRole('textbox', { name: 'Height' }),
    ).toHaveValue('about 1.2 m at the shoulder')
  })
})
