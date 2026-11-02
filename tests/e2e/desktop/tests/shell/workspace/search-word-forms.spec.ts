import type { Locator } from 'playwright/test'
import { test, expect } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import { createNoteWithBody, destroyNotes } from '../../support/layout.js'
import { createUniqueTitle } from '../../support/vault.js'

/** Invented words and a made-up text: the same word in two forms, as Russian declines it. */
const EXACT = 'Moya zheltaya stolba stoit u okna'
const OTHER = 'Kuvshin lezhit u stolbe pod oknom'

test.describe('Search for a word in another form', () => {
  const created: string[] = []

  test.afterEach(async ({ request }) => {
    await destroyNotes(request, created.splice(0))
  })

  const writeBoth = async (request: Parameters<typeof destroyNotes>[0]) => {
    created.push(
      await createNoteWithBody(request, createUniqueTitle('Zapis one'), 'notes', EXACT),
      await createNoteWithBody(request, createUniqueTitle('Zapis two'), 'notes', OTHER),
    )
  }

  const openPalette = async (page: Parameters<typeof openWorkspace>[0]) => {
    await openWorkspace(page)
    await page.keyboard.press('Control+k')
    const field = page.locator('input[data-palette="field"]').first()
    await expect(field).toBeVisible()
    return field
  }

  const texts = (page: Parameters<typeof openWorkspace>[0], group: RegExp) =>
    page.getByRole('group', { name: group }).getByRole('option')

  const marked = async (row: Locator) =>
    (await row.locator('[data-hit]').allTextContents()).join('|')

  test('finds a word typed in the form of one note and in the form of another', async ({
    page,
    request,
  }) => {
    await writeBoth(request)
    const field = await openPalette(page)
    const exact = texts(page, /^Text/).filter({ hasText: 'zheltaya stolba' })
    const other = texts(page, /^Text/).filter({ hasText: 'lezhit u stolbe' })

    // The note that holds the typed form is first, and the form it holds is marked whole.
    await expect(async () => {
      await field.fill('')
      await field.fill('stolba')
      await expect(exact).toHaveCount(1, { timeout: 3_000 })
      await expect(other).toHaveCount(1, { timeout: 3_000 })
    }).toPass({ timeout: 30_000 })
    const rows = texts(page, /^Text/).filter({ hasText: /stolb[ae]/ })
    await expect(rows.first()).toContainText('zheltaya stolba')
    expect(await marked(exact)).toBe('stolba')
    expect(await marked(other)).toBe('stolb')

    // The other form comes first when it is the one typed.
    await field.fill('stolbe')
    await expect(rows.first()).toContainText('lezhit u stolbe')
    await expect(exact).toHaveCount(1)

    // A form that neither note holds finds both, by the stem of the word.
    await field.fill('stolbu')
    await expect(exact).toHaveCount(1)
    await expect(other).toHaveCount(1)
    expect(await marked(exact)).toBe('stolb')
  })

  test('puts the note with the word first among the answers by meaning too', async ({
    page,
    request,
  }) => {
    await writeBoth(request)
    const field = await openPalette(page)
    const meaning = texts(page, /Meaning/).filter({ hasText: /stolb[ae]/ })
    await expect(async () => {
      await field.fill('')
      await field.fill('stolba')
      await expect(meaning).toHaveCount(2, { timeout: 3_000 })
    }).toPass({ timeout: 30_000 })
    await expect(meaning.first()).toContainText('zheltaya stolba')

    await field.fill('stolbe')
    await expect(meaning).toHaveCount(2)
    await expect(meaning.first()).toContainText('lezhit u stolbe')
  })

  test('a word of three letters is a whole word or a prefix and nothing else', async ({
    page,
    request,
  }) => {
    created.push(
      await createNoteWithBody(
        request,
        createUniqueTitle('Zapis dom'),
        'notes',
        'Bolshoy zhuk dom stoit',
      ),
    )
    const field = await openPalette(page)
    await expect(async () => {
      await field.fill('')
      await field.fill('dom')
      await expect(texts(page, /^Text/).filter({ hasText: 'Bolshoy zhuk' })).toHaveCount(1, {
        timeout: 3_000,
      })
    }).toPass({ timeout: 30_000 })
    await field.fill('dum')
    await expect(field).toHaveValue('dum')
    await expect(texts(page, /^Text/).filter({ hasText: 'Bolshoy zhuk' })).toHaveCount(0)
  })
})
