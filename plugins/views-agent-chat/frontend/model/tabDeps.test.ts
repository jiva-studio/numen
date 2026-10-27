/**
 * What an agent tab asks of the window, answered by the host's workspace and agent.
 */
import { describe, expect, it } from 'vitest'
import { createHost } from '../fixtures'
import { createAgentTabDeps } from './tabDeps'

describe('what an agent tab asks of the window', () => {
  it('opens a file with the runs of its text marked', () => {
    const { host, recorded } = createHost()

    createAgentTabDeps(host).openFileAt('Source.pdf', { from: 1, to: 4 }, { from: 9, to: 12 })

    expect(recorded.openedAt).toStrictEqual([
      [
        'Source.pdf',
        [
          { from: 1, to: 4 },
          { from: 9, to: 12 },
        ],
      ],
    ])
  })

  it('opens a file alongside the pane the person is in', () => {
    const { host, recorded } = createHost()

    createAgentTabDeps(host).openFileBeside('Note.md')

    expect(recorded.opened).toStrictEqual([['Note.md', { placement: 'beside' }]])
  })

  it('resolves the wikilinks of an answer through the agent', async () => {
    const { host, recorded } = createHost({ notes: { 'name://Plan': 'Ideas/Plan.md' } })

    const found = await createAgentTabDeps(host).resolveLinks('[[Plan]] and [[Nowhere]]')

    expect([...found]).toStrictEqual([
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
      settings: { 'agent.use': 'zeta' },
      models: { 'agent.zeta.model': [model] },
    })
    const deps = createAgentTabDeps(host)

    await deps.writeSetting?.(['agent', 'use'], 'omega')

    expect(deps.getSetting?.(['agent', 'use'])).toBe('zeta')
    expect(deps.getModelsAt?.(['agent', 'zeta', 'model'])).toStrictEqual([model])
    expect(deps.getModelsAt?.(['agent', 'nobody', 'model'])).toStrictEqual([])
    expect(recorded.settings).toStrictEqual([[['agent', 'use'], 'omega']])
  })
})
