/**
 * What an agent tab requires of the window, served by the host's workspace and the links of the
 * vault.
 */
import { describe, expect, it } from 'vitest'
import { createHost } from '../fixtures'
import { createAgentTabDeps } from './tabDeps'

describe('what an agent tab asks of the window', () => {
  it('opens a file in the pane the person is in', () => {
    const { host, recorded } = createHost()

    createAgentTabDeps(host).openFile('notes/Orchard.md')

    expect(recorded.opened).toStrictEqual([['notes/Orchard.md', undefined]])
  })

  it('opens a file with the runs of its text marked', () => {
    const { host, recorded } = createHost()

    createAgentTabDeps(host).openFileAt('Source.pdf', { start: 1, end: 4 }, { start: 9, end: 12 })

    expect(recorded.openedAt).toStrictEqual([
      [
        'Source.pdf',
        [
          { start: 1, end: 4 },
          { start: 9, end: 12 },
        ],
      ],
    ])
  })

  it('opens a file alongside the pane the person is in', () => {
    const { host, recorded } = createHost()

    createAgentTabDeps(host).openFileAdjacent('Note.md')

    expect(recorded.opened).toStrictEqual([['Note.md', { placement: 'split' }]])
  })

  it('resolves the wikilinks of an answer through the links of the vault', async () => {
    const { host, recorded } = createHost({ notes: { 'name://Plan': 'Ideas/Plan.md' } })

    const found = await createAgentTabDeps(host).resolveLinks('[[Plan]] and [[Nowhere]]')

    expect(found.map((one) => [one.address, one.path])).toStrictEqual([
      ['name://Plan', 'Ideas/Plan.md'],
      ['name://Nowhere', ''],
    ])
    expect(recorded.resolved).toStrictEqual(['[[Plan]] and [[Nowhere]]'])
  })

  it('opens the settings page of the plugin', () => {
    const { host, recorded } = createHost()

    createAgentTabDeps(host).openSettings()

    expect(recorded.settingsOpened).toBe(1)
  })
})
