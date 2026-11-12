/**
 * What a plex tab requests from the window, as answered by the host's workspace
 * and the notes of the vault.
 */
import { describe, expect, it } from 'vitest'
import type { Connection } from '@numen/plugin-sdk'
import { createHost } from '../fixtures/host'
import { createPlexTabDeps } from './tabDeps'

const emptyConnection: Connection = {
  first: 'One.md',
  second: 'Two.md',
  role: null,
  type: '',
  firstSide: null,
  secondSide: null,
}

describe('a note a plex tab opens', () => {
  it('goes to the tabs linked to it, and opens nowhere else', () => {
    const { host, recorded } = createHost({ linked: { 'plex:1': true } })

    createPlexTabDeps(host, 'plex:1').window.openNote('One.md', { title: 'One', showing: 'split' })

    expect(recorded.synced).toStrictEqual([['plex:1', 'One.md', 'One']])
    expect(recorded.opened).toStrictEqual([])
  })

  it('opens at the line requested even where a tab is linked to it', () => {
    const { host, recorded } = createHost({ linked: { 'plex:1': true } })

    createPlexTabDeps(host, 'plex:1').window.openNote('One.md', { title: 'One', line: 4 })

    expect(recorded.synced).toStrictEqual([])
    expect(recorded.opened).toStrictEqual([['One.md', { title: 'One', line: 4 }]])
  })

  it('opens where the person requested, at the line requested, where no tab is linked to it', () => {
    const { host, recorded } = createHost()

    createPlexTabDeps(host, 'plex:1').window.openNote('One.md', {
      title: 'One',
      showing: 'split',
      line: 4,
    })

    expect(recorded.opened).toStrictEqual([
      ['One.md', { title: 'One', placement: 'split', line: 4 }],
    ])
  })

  it('opens with no title, placement or line where none is requested', () => {
    const { host, recorded } = createHost()

    createPlexTabDeps(host, 'plex:1').window.openNote('One.md')

    expect(recorded.synced).toStrictEqual([['plex:1', 'One.md', '']])
    expect(recorded.opened).toStrictEqual([['One.md', { title: '' }]])
  })
})

describe('what else a plex tab asks of the window', () => {
  it('moves the tabs linked to it to a path it selects', () => {
    const { host, recorded } = createHost()
    const deps = createPlexTabDeps(host, 'plex:2')

    deps.window.onSelectPath?.('Two.md', 'Two')
    deps.window.onSelectPath?.('Three.md')

    expect(recorded.synced).toStrictEqual([
      ['plex:2', 'Two.md', 'Two'],
      ['plex:2', 'Three.md', ''],
    ])
  })

  it('reads the entries of the node menu from its slot, runs a command over a note and says what failed', async () => {
    const { host, recorded } = createHost()
    const deps = createPlexTabDeps(host, 'plex:1')
    const target = { path: 'One.md', title: 'One', isFolder: false, others: [], tab: '' }

    deps.menu.getMenuEntries(target)
    await deps.menu.runCommand('rename', target)
    deps.window.showMessage('the link was not written')

    expect(recorded.entries).toStrictEqual([['plexGraph.context', target]])
    expect(recorded.commands).toStrictEqual([['rename', target]])
    expect(recorded.errors).toStrictEqual(['the link was not written'])
  })

  it('makes and joins notes through the notes of the vault', async () => {
    const { host, recorded } = createHost()
    const deps = createPlexTabDeps(host, 'plex:1')

    await deps.editor.createRelated({ from: 'One.md', role: 'child' })
    expect(
      await deps.editor.createRelated({ from: 'One.md', role: 'parent', title: 'Two' }),
    ).toStrictEqual({ ok: true, value: 'Two.md' })
    expect(await deps.editor.join('One.md', 'Two.md', { role: 'jump' })).toStrictEqual({
      ok: true,
      value: undefined,
    })
    expect(await deps.editor.getConnection('One.md', 'Two.md')).toMatchObject({ ok: true })
    expect(await deps.editor.saveConnection(emptyConnection)).toStrictEqual({
      ok: true,
      value: undefined,
    })
    expect(await deps.editor.removeConnection('One.md', 'Two.md')).toStrictEqual({
      ok: true,
      value: undefined,
    })

    expect(recorded.made).toStrictEqual([
      ['One.md', 'child'],
      ['One.md', 'parent'],
    ])
    expect(recorded.joined).toStrictEqual([['One.md', 'Two.md', 'jump']])
    expect(deps.creatable).toStrictEqual(['parent', 'child', 'jump'])
  })

  it('reads what the window shows, and asks the vault where it opens', async () => {
    const { host, requested } = createHost({ opening: 'Root.md' })
    const deps = createPlexTabDeps(host, 'plex:1')

    expect(deps.window.isReady.value).toBe(true)
    expect(deps.window.openingPath.value).toBe('Root.md')
    expect(deps.notes.isNesting.value).toBe(true)
    expect(deps.notes.headings.value).toBe(6)
    expect(deps.window.dragged.value).toStrictEqual([])
    expect(await deps.window.readOpeningPath()).toBe('Root.md')
    await deps.notes.readHeadings(['Root.md'])
    expect(requested.headings).toStrictEqual([['Root.md']])
    expect(await deps.notes.searchNames?.('Ro', 5)).toStrictEqual([])
  })
})
