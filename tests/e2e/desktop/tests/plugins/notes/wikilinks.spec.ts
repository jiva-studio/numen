import path from 'node:path'
import { test, expect, readActivePath } from '../../support/test.js'
import {
  getGraph,
  getPaletteField,
  openWorkspace,
  runPaletteCommand,
} from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  getNoteTab,
  getNoteText,
  listBacklinks,
  listFilePaths,
  openNote,
  readNote,
  typeAtEnd,
  waitForSavedBody,
  writeNote,
} from '../../support/vault.js'

test.describe('Note Creation and Wikilinks', () => {
  const madeNotes: string[] = []

  test.afterEach(async ({ request }) => {
    for (const one of madeNotes.splice(0)) await destroyNote(request, one)
  })

  test('makes a note on disk with Create note and opens it in front', async ({ page, request }) => {
    const title = createUniqueTitle('Quantum')
    await openWorkspace(page)

    await runPaletteCommand(page, 'Create note', 'Create note')
    await expect(page.locator('[data-palette="crumb"]')).toBeVisible()
    await getPaletteField(page).fill(title)
    await expect(page.getByRole('option', { selected: true })).toContainText(title)
    await page.keyboard.press('Enter')

    await expect.poll(() => readActivePath(request)).toMatch(new RegExp(`${title}\\.md$`))
    const made = await readActivePath(request)
    madeNotes.push(made)
    await expect(
      page.getByRole('tab', { name: `${title} Close ${title}`, exact: true }),
    ).toHaveAttribute('aria-selected', 'true')
    await expect(getGraph(page).getByRole('img', { name: `${title}, focus` })).toBeAttached()
    expect(await listFilePaths(request, path.posix.dirname(made).replace(/^\.$/, ''))).toContain(
      made,
    )
    expect((await readNote(request, made)).error).toBe('')
  })

  test('completes a typed wikilink from the notes of the vault, and the core reads it as a link', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Heat engine'))
    try {
      await writeNote(request, notePath, 'Work from a difference in temperature.\n')
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text).toContainText('Work from a difference in temperature.')

      await typeAtEnd(page, text, '\nSee [[Entr')
      const completion = page
        .locator('.cm-tooltip-autocomplete')
        .getByRole('option', {
          name: /^Entropy/,
        })
        .first()
      await expect(completion).toBeVisible()
      await completion.click()

      await waitForSavedBody(request, notePath, 'See [[Entropy]]')
      await expect.poll(() => listBacklinks(request, 'notes/Entropy.md')).toContain(notePath)
    } finally {
      await destroyNote(request, notePath)
    }
  })

  test('opens the note a wikilink names when it is clicked with Ctrl', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Carnot cycle'))
    try {
      await writeNote(request, notePath, 'The cycle is bounded by [[Entropy]] alone.\n')
      await openWorkspace(page)
      const text = await openNote(page, notePath)

      await text
        .getByText('Entropy', { exact: false })
        .first()
        .click({ modifiers: ['ControlOrMeta'] })

      await expect(getNoteTab(page, 'notes/Entropy.md')).toHaveAttribute('aria-selected', 'true')
      await expect(getNoteText(page, 'notes/Entropy.md')).toContainText('A measure of uncertainty')
    } finally {
      await destroyNote(request, notePath)
    }
  })
})

test.describe('Wikilink in the editor', () => {
  test('draws a finished wikilink as one link and opens its note from the shown text', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Link design'))
    try {
      await writeNote(request, notePath, 'See\nnext line')
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text.locator('.cm-line:nth-child(2)')).toHaveText('next line')

      // The completion list of `[[Entr` answers from the vault index and can open after the
      // last key: the caret leaves the line with a key the list does not take.
      await text.click()
      await page.keyboard.press('ControlOrMeta+Home')
      await page.keyboard.press('End')
      await page.keyboard.type(' [[Entropy]]')
      await page.keyboard.press('ControlOrMeta+End')

      await expect(text.locator('.cm-line').first()).toHaveText('See Entropy')
      await expect(text.locator('.cm-line:nth-child(2)')).toHaveText('next line')
      await expect(text.locator('.cm-wikilink')).toHaveText('Entropy')
      await expect(text.locator('.cm-wikilink-mark')).toHaveCount(0)

      await text.locator('.cm-wikilink').click({ modifiers: ['ControlOrMeta'] })

      await expect(getNoteTab(page, 'notes/Entropy.md')).toHaveAttribute('aria-selected', 'true')
    } finally {
      await destroyNote(request, notePath)
    }
  })
})
