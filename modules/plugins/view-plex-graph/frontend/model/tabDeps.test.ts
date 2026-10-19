/**
 * What a plex tab asks of the window, as the host's workspace and the notes of
 * the vault answer it.
 */
import { describe, expect, it } from 'vitest'
import { createHost } from '../fixtures/host'
import { createPlexTabDeps } from './tabDeps'

describe('a note a plex tab opens', () => {
  it('goes to the tabs linked to it, and opens nowhere else', () => {
    const { host, recorded } = createHost({ linked: { 'plex:1': true } })

    createPlexTabDeps(host, 'plex:1').openNote('One.md', { title: 'One', showing: 'beside' })

    expect(recorded.synced).toStrictEqual([['plex:1', 'One.md', 'One']])
    expect(recorded.opened).toStrictEqual([])
  })

  it('opens where the person asked, at the line asked, where no tab is linked to it', () => {
    const { host, recorded } = createHost()

    createPlexTabDeps(host, 'plex:1').openNote('One.md', {
      title: 'One',
      showing: 'beside',
      line: 4,
    })

    expect(recorded.opened).toStrictEqual([
      ['One.md', { title: 'One', placement: 'beside', line: 4 }],
    ])
  })

  it('opens with no title, placement or line where none is asked', () => {
    const { host, recorded } = createHost()

    createPlexTabDeps(host, 'plex:1').openNote('One.md')

    expect(recorded.synced).toStrictEqual([['plex:1', 'One.md', '']])
    expect(recorded.opened).toStrictEqual([['One.md', { title: '' }]])
  })
})

describe('what else a plex tab asks of the window', () => {
  it('moves the tabs linked to it to a path it selects', () => {
    const { host, recorded } = createHost()
    const deps = createPlexTabDeps(host, 'plex:2')

    deps.onSelectPath?.('Two.md', 'Two')
    deps.onSelectPath?.('Three.md')

    expect(recorded.synced).toStrictEqual([
      ['plex:2', 'Two.md', 'Two'],
      ['plex:2', 'Three.md', ''],
    ])
  })

  it('runs a command over a note, asks the agent and says what failed', () => {
    const { host, recorded } = createHost()
    const deps = createPlexTabDeps(host, 'plex:1')

    deps.runCommand('rename', 'One.md', 'One')
    deps.askAgent('what is One?')
    deps.showMessage('the link was not written')

    expect(recorded.commands).toStrictEqual([['rename', { path: 'One.md', title: 'One' }]])
    expect(recorded.questions).toStrictEqual(['what is One?'])
    expect(recorded.errors).toStrictEqual(['the link was not written'])
  })

  it('makes and joins notes through the notes of the vault', async () => {
    const { host, recorded } = createHost()
    const deps = createPlexTabDeps(host, 'plex:1')

    await deps.editor.createInRelation('One.md', 'child')
    expect(await deps.editor.createWithTitle('Two', 'One.md', 'parent')).toStrictEqual({
      path: 'Two.md',
      title: 'Two',
    })
    expect(await deps.editor.join('One.md', 'Two.md', 'jump')).toBe(true)
    expect(await deps.editor.replaceLinks('One.md', 'Two.md', [])).toBe(true)
    expect(await deps.createUntitledNote()).toBe('Untitled.md')

    expect(recorded.made).toStrictEqual([
      ['One.md', 'child'],
      ['One.md', 'parent'],
    ])
    expect(recorded.joined).toStrictEqual([['One.md', 'Two.md', 'jump']])
    expect(deps.creatable).toStrictEqual(['parent', 'child', 'jump'])
  })

  it('reads what the window shows, and asks the vault where it opens', async () => {
    const { host, asked } = createHost({ opening: 'Root.md' })
    const deps = createPlexTabDeps(host, 'plex:1')

    expect(deps.isReady.value).toBe(true)
    expect(deps.openingPath.value).toBe('Root.md')
    expect(deps.isNesting.value).toBe(true)
    expect(deps.headings.value).toBe(6)
    expect(deps.dragged.value).toStrictEqual([])
    expect(await deps.readOpeningPath()).toBe('Root.md')
    await deps.readHeadings(['Root.md'])
    expect(asked.headings).toStrictEqual([['Root.md']])
    expect(await deps.searchNames?.('Ro', 5)).toStrictEqual([])
  })
})
