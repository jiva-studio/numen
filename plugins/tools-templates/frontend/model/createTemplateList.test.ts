import { describe, expect, it } from 'vitest'
import type { VaultEntry } from '@numen/plugin-sdk'
import { createTemplateList, type TemplateFiles } from './createTemplateList'

const note = (path: string): VaultEntry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  isNote: true,
  type: 'note',
})

const file = (path: string): VaultEntry => ({ ...note(path), isNote: false, type: '' })

const folder = (path: string): VaultEntry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: true,
  isNote: false,
  type: '',
})

const createVault = (folders: Record<string, readonly VaultEntry[] | Error>) => {
  const listed: string[] = []
  const files: TemplateFiles = {
    list: async (path) => {
      listed.push(path)
      const found = folders[path]
      if (found === undefined || found instanceof Error) {
        throw found ?? new Error(`no folder ${path}`)
      }
      return found
    },
  }
  return { files, listed }
}

const vault = {
  Templates: [
    note('Templates/Meeting.md'),
    note('Templates/Book review.md'),
    file('Templates/diagram.png'),
    file('Templates/todo.txt'),
    folder('Templates/Daily'),
  ],
  'Templates/Daily': [note('Templates/Daily/Journal.md')],
  Models: [note('Models/Recipe.md')],
}

describe('the template list', () => {
  it('lists the notes of the folder in the setting', async () => {
    const { files } = createVault(vault)
    const list = createTemplateList({ files, getFolder: () => 'Templates' })

    const items = await list.listItems()

    expect(items.map((item) => [item.id, item.title, item.group])).toEqual([
      ['Templates/Book review.md', 'Book review', undefined],
      ['Templates/Meeting.md', 'Meeting', undefined],
      ['Templates/Daily/Journal.md', 'Journal', 'Daily'],
    ])
  })

  it('lists other notes after the setting changes', async () => {
    const { files } = createVault(vault)
    let setting = 'Templates'
    const list = createTemplateList({ files, getFolder: () => setting })

    const before = await list.listItems()
    setting = 'Models'
    const after = await list.listItems()

    expect(before.map((item) => item.id)).toContain('Templates/Meeting.md')
    expect(after.map((item) => item.id)).toEqual(['Models/Recipe.md'])
  })

  it('lists the notes of nested subfolders in the group of their path', async () => {
    const { files } = createVault({
      Templates: [folder('Templates/Meetings')],
      'Templates/Meetings': [
        folder('Templates/Meetings/Weekly'),
        note('Templates/Meetings/One to one.md'),
      ],
      'Templates/Meetings/Weekly': [note('Templates/Meetings/Weekly/Standup.md')],
    })

    const items = await createTemplateList({ files, getFolder: () => 'Templates' }).listItems()

    expect(items.map((item) => [item.title, item.group])).toEqual([
      ['One to one', 'Meetings'],
      ['Standup', 'Meetings/Weekly'],
    ])
  })

  it('visits each folder once', async () => {
    const { files, listed } = createVault(vault)

    await createTemplateList({ files, getFolder: () => 'Templates' }).listItems()

    expect([...listed].sort()).toEqual(['Templates', 'Templates/Daily'])
  })

  it('skips files that are not notes', async () => {
    const { files } = createVault({
      Templates: [
        file('Templates/diagram.png'),
        file('Templates/todo.txt'),
        note('Templates/Plan.md'),
      ],
    })

    const items = await createTemplateList({ files, getFolder: () => 'Templates' }).listItems()

    expect(items.map((item) => item.id)).toEqual(['Templates/Plan.md'])
  })

  it('skips a subfolder that cannot be listed', async () => {
    const { files } = createVault({
      Templates: [note('Templates/Plan.md'), folder('Templates/Broken'), folder('Templates/Daily')],
      'Templates/Broken': new Error('access denied'),
      'Templates/Daily': [note('Templates/Daily/Journal.md')],
    })

    const items = await createTemplateList({ files, getFolder: () => 'Templates' }).listItems()

    expect(items.map((item) => item.id)).toEqual([
      'Templates/Plan.md',
      'Templates/Daily/Journal.md',
    ])
  })

  it('lists nothing when the folder is missing', async () => {
    const { files } = createVault({})

    const items = await createTemplateList({ files, getFolder: () => 'Templates' }).listItems()

    expect(items).toEqual([])
  })

  it('lists nothing while the folder holds no note', async () => {
    const { files } = createVault({ Templates: [folder('Templates/Daily')], 'Templates/Daily': [] })

    const items = await createTemplateList({ files, getFolder: () => 'Templates' }).listItems()

    expect(items).toEqual([])
  })
})
