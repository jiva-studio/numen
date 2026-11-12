/** Command execution against a window double: the registry, the commands of the palette, and the settings. */
import { beforeEach, describe, expect, it } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { ReportedError } from '@numen/core-client'
import {
  createCommandRegistry,
  type CommandDeps,
  type CommandInvocation,
  type CommandTarget,
} from '@/shared/model/command'
import { createInvocation } from '../../lib/entries/invocation'
import { runSupport } from '../palette/runSupport'
import { runInvocation } from './handlers'

const front = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'plex',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...override,
})

let registry = createCommandRegistry()

beforeEach(() => {
  registry = createCommandRegistry()
})

const window = (override: Partial<CommandDeps> = {}) => {
  const actions: string[] = []
  const notices: string[] = []
  const severities: string[] = []
  /** Every call to the voice, the empty ones and the missing severities included. */
  const calls: [string | undefined, string | undefined][] = []
  const on: CommandDeps = {
    notes: {
      getTabAt: () => null,
      getPath: (id) => (id === 'member' ? 'physics/Being.md' : id),
      asking: () => false,
      flushPending: async () => undefined,
      close: () => undefined,
      openFile: () => undefined,
    },
    navigator: {
      openTab: (kind) => {
        actions.push(`openTab ${kind}`)
      },
      closeTab: (tab) => {
        actions.push(`closeTab ${tab}`)
      },
      search: () => {
        actions.push('search')
      },
    },
    settings: {
      chooseSetting: async (command, selected) => {
        actions.push(`setting ${command} ${selected}`)
      },
    },
    runSupport: runSupport(),
    registry,
    writeMessage: (text, kind) => {
      calls.push([text, kind])
      if (!text) return
      notices.push(text)
      severities.push(kind ?? '')
    },
    ...override,
  }
  return { on, actions, notices, severities, calls }
}

const carry = async (invocation: CommandInvocation | null, on: CommandDeps) =>
  runInvocation(invocation, on)

describe('the invocation of a command', () => {
  it('emits onCommand:<id> when a command is executed', async () => {
    const events: string[] = []
    const one = window({ emitEvent: (event) => events.push(event) })
    registry.register('app:file.open', () => undefined)

    await carry(createInvocation('app:file.open', front()), one.on)

    expect(events).toStrictEqual(['onCommand:app:file.open'])
  })

  it('clears the message of the last command before it runs', async () => {
    const one = window()
    registry.register('app:file.open', () => undefined)

    await carry(createInvocation('app:file.open', front()), one.on)

    expect(one.calls).toStrictEqual([['', undefined]])
  })

  it('answers the path a command made, and nothing for a command that made none', async () => {
    const one = window()
    registry.register('makes', () => 'Entropy.md')
    registry.register('makes-none', () => undefined)

    expect(await carry(createInvocation('makes', front()), one.on)).toBe('Entropy.md')
    expect(await carry(createInvocation('makes-none', front()), one.on)).toBe('')
  })
})

describe('pluggable command registry', () => {
  it('registers dynamic commands and carries them out', async () => {
    const executed: string[] = []
    const unregister = registry.register('customAction', (invocation) => {
      executed.push(`custom:${invocation.target.path}`)
    })

    const one = window()
    await carry(createInvocation('customAction', front()), one.on)
    expect(executed).toStrictEqual(['custom:physics/Ontology.md'])

    unregister()
    await carry(createInvocation('customAction', front()), one.on)
    expect(executed).toHaveLength(1)
  })

  it('allows overriding the commands of the palette via the registry', async () => {
    const executed: string[] = []
    const unregister = registry.register('app:tab.close', () => {
      executed.push('custom-close')
    })

    const one = window()
    await carry(createInvocation('app:tab.close', front()), one.on)
    expect(executed).toStrictEqual(['custom-close'])
    expect(one.actions).toStrictEqual([])

    unregister()
    await carry(createInvocation('app:tab.close', front()), one.on)
    expect(one.actions).toStrictEqual(['closeTab tab'])
  })

  it('does not delete a newer handler when an older unregister function is called', () => {
    const firstUnregister = registry.register('dup', () => undefined)
    const secondHandler = () => undefined
    const secondUnregister = registry.register('dup', secondHandler)

    firstUnregister()
    expect(registry.get('dup')).toBe(secondHandler)

    secondUnregister()
    expect(registry.get('dup')).toBeUndefined()
  })
})

describe('a command over the window', () => {
  it('opens the settings, and closes the tab in front', async () => {
    const one = window()

    await carry(createInvocation('app:settings.open', front()), one.on)
    await carry(createInvocation('app:tab.close', front()), one.on)

    expect(one.actions).toStrictEqual(['openTab settings', 'closeTab tab'])
  })

  it('passes the field back to the search', async () => {
    const one = window()

    await carry(createInvocation('app:vault.search', front()), one.on)

    expect(one.actions).toStrictEqual(['search'])
  })
})

describe('a note that moved between the answer and the invocation', () => {
  it('is carried out over its current position, not the one the invocation was made over', async () => {
    const one = window()
    const paths: string[] = []
    registry.register('over', (invocation) => {
      paths.push(invocation.target.path)
    })

    await carry(createInvocation('over', front(), '', 'member'), one.on)

    expect(paths).toStrictEqual(['physics/Being.md'])
  })

  it('is left at the position it was made over where no tab holds it', async () => {
    const one = window()
    const paths: string[] = []
    registry.register('over', (invocation) => {
      paths.push(invocation.target.path)
    })

    await carry(createInvocation('over', front()), one.on)

    expect(paths).toStrictEqual(['physics/Ontology.md'])
  })
})

describe('nothing to carry out', () => {
  it('does nothing at all', async () => {
    const one = window()

    await carry(null, one.on)
    await carry(createInvocation('constructor', front()), one.on)
    await carry(createInvocation('', front()), one.on)

    expect(one.actions).toStrictEqual([])
  })

  it('does nothing and says nothing for a command nothing carries out', async () => {
    const one = window()

    await carry({ ...createInvocation('app:file.open', front()), id: 'nothing' }, one.on)

    expect(one.actions).toStrictEqual([])
    expect(one.calls).toStrictEqual([])
  })

  it('says what a command could not request of the core, and asks no further', async () => {
    const one = window()
    registry.register('app:file.remove', () => Promise.reject(createLostConnection('gone')))

    await carry(createInvocation('app:file.remove', front()), one.on)

    expect(one.notices).toStrictEqual(['Numen is not responding; retrying'])
    expect(one.severities).toStrictEqual(['error'])
  })
})

describe('the command of a setting', () => {
  it('passes the command and the row chosen to the adapter of the settings, whoever owns the setting', async () => {
    const one = window()

    await carry(createInvocation('setting:app:appearance.theme', front(), 'mine:sea'), one.on)
    await carry(
      createInvocation('setting:md.numen.engines.transcription:autoRun', front(), 'on'),
      one.on,
    )

    expect(one.actions).toStrictEqual([
      'setting setting:app:appearance.theme mine:sea',
      'setting setting:md.numen.engines.transcription:autoRun on',
    ])
  })

  it('passes the row chosen whatever note is in front', async () => {
    const one = window()

    await carry(createInvocation('setting:app:performance.profile', front(), 'eco'), one.on)

    expect(one.actions).toStrictEqual(['setting setting:app:performance.profile eco'])
  })

  it('shows the text of a ReportedError from a setting write as an error notice', async () => {
    const one = window({
      settings: {
        chooseSetting: async () => {
          throw new ReportedError('value out of range')
        },
      },
    })

    await carry(createInvocation('setting:app:performance.profile', front(), 'eco'), one.on)

    expect(one.notices).toStrictEqual(['value out of range'])
    expect(one.severities).toStrictEqual(['error'])
  })

  it('shows the generic text for a setting write that rejects with a plain Error', async () => {
    const one = window({
      settings: {
        chooseSetting: async () => {
          throw new Error('settings.json: EACCES')
        },
      },
    })

    await carry(createInvocation('setting:app:performance.profile', front(), 'eco'), one.on)

    expect(one.notices).toStrictEqual(['Numen is not responding; retrying'])
    expect(one.severities).toStrictEqual(['error'])
  })
})
