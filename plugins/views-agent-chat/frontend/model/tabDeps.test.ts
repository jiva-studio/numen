/**
 * What an agent tab requires of the window, served by the host's workspace and
 * agent.
 */
import { describe, expect, it } from 'vitest'
import { createHost } from '../fixtures'
import { createAgentTabDeps } from './tabDeps'

describe('what an agent tab asks of the window', () => {
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

  it('resolves the wikilinks of an answer through the agent', async () => {
    const { host, recorded } = createHost({ notes: { 'name://Plan': 'Ideas/Plan.md' } })

    const found = await createAgentTabDeps(host).resolveLinks('[[Plan]] and [[Nowhere]]')

    expect(found.map((one) => [one.address, one.path])).toStrictEqual([
      ['name://Plan', 'Ideas/Plan.md'],
      ['name://Nowhere', ''],
    ])
    expect(recorded.resolved).toStrictEqual(['[[Plan]] and [[Nowhere]]'])
  })

  it('says why the agent cannot be reached', () => {
    const { host } = createHost({ unreachable: 'the agent is down' })

    expect(createAgentTabDeps(host).unreachable()).toBe('the agent is down')
  })

  it('reads and writes the settings, and lists the models, through the agent', async () => {
    const model = { name: 'small', title: 'Small', isDefault: true, presence: 'present' } as const
    const { host, recorded } = createHost({
      settings: { 'app.agent.use': 'zeta' },
      models: { 'app.agent.zeta.model': [model] },
    })
    const deps = createAgentTabDeps(host)

    await deps.settings?.writeSetting?.(['app', 'agent', 'use'], 'omega')

    expect(deps.settings?.getSetting?.(['app', 'agent', 'use'])).toBe('zeta')
    expect(deps.settings?.getModelsAt?.(['app', 'agent', 'zeta', 'model'])).toStrictEqual([model])
    expect(deps.settings?.getModelsAt?.(['app', 'agent', 'nobody', 'model'])).toStrictEqual([])
    expect(recorded.settings).toStrictEqual([[['app', 'agent', 'use'], 'omega']])
  })
})

describe('opening the settings from an agent tab', () => {
  it('runs the command of the window that opens the settings, over no target', () => {
    const { host, recorded } = createHost({ focus: { path: 'Orchard.md', title: 'Orchard' } })

    createAgentTabDeps(host).settings?.openSettings?.()

    expect(recorded.commands).toStrictEqual([['app:settings.open', undefined]])
  })

  it('opens no settings in a window that offers no commands', () => {
    const { host, recorded } = createHost()
    const { commands: _commands, ...rest } = host

    createAgentTabDeps(rest).settings?.openSettings?.()

    expect(recorded.commands).toStrictEqual([])
  })
})
