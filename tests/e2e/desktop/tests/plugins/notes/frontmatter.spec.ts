import { test, expect } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  getNotePanel,
  getNoteText,
  openNote,
  readNote,
  readVaultFile,
  typeAtEnd,
  waitForSavedBody,
  writeNote,
  writeVaultFile,
} from '../../support/vault.js'

/** A file the note tab opens: its frontmatter as written, and the prose the editor must show. */
interface Shape {
  readonly name: string
  readonly frontmatter: (id: string) => string
  readonly prose: string
  readonly visible: string
}

const SHAPES: readonly Shape[] = [
  {
    name: 'an empty body',
    frontmatter: (id) => `---\nid: ${id}\n---\n`,
    prose: '',
    visible: 'Write',
  },
  {
    name: 'a body that starts with an unfinished wikilink',
    frontmatter: (id) => `---\nid: ${id}\n---\n`,
    prose: '[[B',
    visible: '[[B',
  },
  {
    name: 'a frontmatter of several keys',
    frontmatter: (id) => `---\nid: ${id}\ntags: [alpha, beta]\nsource: Test\n---\n`,
    prose: 'Body line',
    visible: 'Body line',
  },
  {
    name: 'a file with CRLF line ends',
    frontmatter: (id) => `---\r\nid: ${id}\r\n---\r\n`,
    prose: 'Body line',
    visible: 'Body line',
  },
  {
    name: 'a frontmatter of an id only, and prose',
    frontmatter: (id) => `---\nid: ${id}\n---\n`,
    prose: 'Prose under the id.',
    visible: 'Prose under the id.',
  },
]

const readId = async (
  request: Parameters<typeof readVaultFile>[0],
  notePath: string,
): Promise<string> => String(/^id: (\S+)/m.exec(await readVaultFile(request, notePath))?.[1])

test.describe('Note editor and the frontmatter of the file', () => {
  for (const shape of SHAPES) {
    test(`shows only the prose and keeps the frontmatter on save: ${shape.name}`, async ({
      page,
      request,
    }) => {
      const notePath = await createNote(request, createUniqueTitle('Frontmatter shape'))
      try {
        const frontmatter = shape.frontmatter(await readId(request, notePath))
        await writeVaultFile(request, notePath, `${frontmatter}${shape.prose}`)
        await openWorkspace(page)
        const text = await openNote(page, notePath)

        await expect(text).toHaveText(shape.visible)
        await expect(getNotePanel(page, notePath).getByRole('alert')).toHaveCount(0)

        await typeAtEnd(page, text, ' tail')

        await waitForSavedBody(request, notePath, 'tail')
        expect((await readNote(request, notePath)).error).toBe('')
        expect(await readVaultFile(request, notePath)).toContain(`${frontmatter}${shape.prose}`)
      } finally {
        await destroyNote(request, notePath)
      }
    })
  }

  test('shows the new prose alone when another program rewrites the open, unedited note', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Frontmatter rewrite'))
    try {
      await writeNote(request, notePath, 'Old prose.')
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text).toHaveText('Old prose.')

      await writeNote(request, notePath, 'New prose.')

      await expect(getNoteText(page, notePath)).toHaveText('New prose.')
      await expect(getNotePanel(page, notePath).getByRole('alert')).toHaveCount(0)
    } finally {
      await destroyNote(request, notePath)
    }
  })

  test('shows no frontmatter in a note made new and typed into before any prose exists', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Frontmatter fresh'))
    try {
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text).toHaveText('Write')

      await typeAtEnd(page, text, '[[B')

      await waitForSavedBody(request, notePath, '[[B')
      await expect(text).toHaveText('[[B')
      await expect(getNotePanel(page, notePath).getByRole('alert')).toHaveCount(0)
    } finally {
      await destroyNote(request, notePath)
    }
  })
})
