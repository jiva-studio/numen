import { describe, expect, it } from 'vitest'
import type { VaultEntry } from '@numen/plugin-sdk'
import { parseMasks } from '../lib/masks'
import { createTemplateList, TemplateListError, type TemplateFiles } from './createTemplateList'

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
      if (found === undefined) {
        throw Object.assign(new Error(`no folder ${path}`), { code: 'not_found' })
      }
      if (found instanceof Error) throw found
      return found
    },
  }
  return { files, listed }
}

const vault = {
  '': [folder('Templates'), folder('Models'), folder('Notes'), note('Inbox.md')],
  Templates: [
    note('Templates/Meeting.md'),
    note('Templates/Book review.md'),
    file('Templates/diagram.png'),
    file('Templates/todo.txt'),
    folder('Templates/Daily'),
  ],
  'Templates/Daily': [note('Templates/Daily/Journal.md')],
  Models: [note('Models/Recipe.md'), note('Models/Weekly.template.md')],
  Notes: [note('Notes/Plan.md'), note('Notes/Standup.template.md')],
}

const listWith = (files: TemplateFiles, raw: string) =>
  createTemplateList({ files, getMasks: () => parseMasks(raw) }).listItems()

describe('the template list', () => {
  it('lists the notes the masks name, grouped by the folder below the static folder', async () => {
    const { files } = createVault(vault)

    const items = await listWith(files, 'Templates/**')

    expect(items.map((item) => [item.id, item.title, item.group])).toEqual([
      ['Templates/Book review.md', 'Book review', undefined],
      ['Templates/Meeting.md', 'Meeting', undefined],
      ['Templates/Daily/Journal.md', 'Journal', 'Daily'],
    ])
  })

  it('lists the notes named *.template.md anywhere, titled without the suffix, in their own folder', async () => {
    const { files } = createVault(vault)

    const items = await listWith(files, '**/*.template.md')

    expect(items.map((item) => [item.id, item.title, item.group])).toEqual([
      ['Models/Weekly.template.md', 'Weekly', 'Models'],
      ['Notes/Standup.template.md', 'Standup', 'Notes'],
    ])
  })

  it('lists the templates of every mask once', async () => {
    const { files } = createVault({
      ...vault,
      Templates: [note('Templates/Plan.template.md')],
      'Templates/Daily': [],
    })

    const items = await listWith(files, 'Templates/**, **/*.template.md')

    expect(items.map((item) => item.id)).toEqual([
      'Templates/Plan.template.md',
      'Models/Weekly.template.md',
      'Notes/Standup.template.md',
    ])
  })

  it('lists other notes after the masks change', async () => {
    const { files } = createVault(vault)
    let setting = 'Templates/**'
    const list = createTemplateList({ files, getMasks: () => parseMasks(setting) })

    const before = await list.listItems()
    setting = 'Models/*'
    const after = await list.listItems()

    expect(before.map((item) => item.id)).toContain('Templates/Meeting.md')
    expect(after.map((item) => item.id)).toEqual(['Models/Recipe.md', 'Models/Weekly.template.md'])
  })

  it('lists the notes of nested subfolders in the group of their path', async () => {
    const { files } = createVault({
      '': [folder('Templates')],
      Templates: [folder('Templates/Meetings')],
      'Templates/Meetings': [
        folder('Templates/Meetings/Weekly'),
        note('Templates/Meetings/One to one.md'),
      ],
      'Templates/Meetings/Weekly': [note('Templates/Meetings/Weekly/Standup.md')],
    })

    const items = await listWith(files, 'Templates/**')

    expect(items.map((item) => [item.title, item.group])).toEqual([
      ['One to one', 'Meetings'],
      ['Standup', 'Meetings/Weekly'],
    ])
  })

  it('visits only the folders a mask could reach, each once', async () => {
    const { files, listed } = createVault(vault)

    await listWith(files, 'Templates/**')

    expect([...listed].sort()).toEqual(['', 'Templates', 'Templates/Daily'])
  })

  it('visits every folder for a mask that starts with **', async () => {
    const { files, listed } = createVault(vault)

    await listWith(files, '**/*.template.md')

    expect([...listed].sort()).toEqual(['', 'Models', 'Notes', 'Templates', 'Templates/Daily'])
  })

  it('skips files that are not notes', async () => {
    const { files } = createVault({
      '': [folder('Templates')],
      Templates: [
        file('Templates/diagram.png'),
        file('Templates/todo.txt'),
        note('Templates/Plan.md'),
      ],
    })

    const items = await listWith(files, 'Templates/**')

    expect(items.map((item) => item.id)).toEqual(['Templates/Plan.md'])
  })

  it('skips a subfolder that is gone', async () => {
    const { files } = createVault({
      '': [folder('Templates')],
      Templates: [note('Templates/Plan.md'), folder('Templates/Gone'), folder('Templates/Daily')],
      'Templates/Daily': [note('Templates/Daily/Journal.md')],
    })

    const items = await listWith(files, 'Templates/**')

    expect(items.map((item) => item.id)).toEqual([
      'Templates/Plan.md',
      'Templates/Daily/Journal.md',
    ])
  })

  it('fails with the folder it could not list when access is denied', async () => {
    const { files } = createVault({
      '': [folder('Templates')],
      Templates: [note('Templates/Plan.md'), folder('Templates/Broken')],
      'Templates/Broken': new Error('access denied'),
    })

    await expect(listWith(files, 'Templates/**')).rejects.toThrow(TemplateListError)
    await expect(listWith(files, 'Templates/**')).rejects.toThrow(
      /Templates\/Broken.*access denied/,
    )
  })

  it('lists nothing when the vault cannot be listed', async () => {
    const { files } = createVault({})

    expect(await listWith(files, 'Templates/**')).toEqual([])
  })

  it('lists nothing while no note is named', async () => {
    const { files } = createVault({
      '': [folder('Templates')],
      Templates: [folder('Templates/Daily')],
      'Templates/Daily': [],
    })

    expect(await listWith(files, 'Templates/**')).toEqual([])
  })
})
