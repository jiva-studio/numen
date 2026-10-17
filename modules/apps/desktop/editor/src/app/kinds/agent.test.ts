/** What the agent conversation is handed by the window. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createAgentKind, type AgentKindDeps } from './agent'

type Options = Record<string, (...args: unknown[]) => unknown>

const h = vi.hoisted(() => ({
  open: null as unknown as () => unknown,
  options: null as unknown,
}))

vi.mock('@numen/ui', () => ({ useConversation: () => ({ name: 'conversation' }) }))
vi.mock('@/entities/tab', () => ({ CONVERSATION: 'conversation', generateId: (p: string) => p }))
vi.mock('@/pages/agent-chat', () => ({
  core: { name: 'agent core' },
  WORDS: { name: 'talk' },
  useAgentConversation: (_conversation: unknown, options: unknown) => {
    h.options = options
    return { name: 'agent conversation' }
  },
  createAgentKind: (_handle: unknown, createState: () => unknown) => {
    h.open = createState
    return {}
  },
}))

const setup = () => {
  const calls: unknown[][] = []
  const record =
    (name: string, answer?: unknown) =>
    (...args: unknown[]) => {
      calls.push([name, ...args])
      return answer
    }
  const models = ref([{ id: 'model' }])
  const unreachable = ref('down')
  createAgentKind({
    core: { resolve: record('resolve', 'Resolved.md') },
    tabOpeners: {
      openFile: record('openFile'),
      openFileAt: record('openFileAt'),
    },
    held: { handle: {} },
    window: { unreachable },
    settings: {
      rest: {
        writeSetting: record('writeSetting', 'written'),
        getSetting: record('getSetting', 'value'),
        getModelsAt: record('getModelsAt', models.value),
      },
    },
    about: () => ({ path: '', title: '' }),
  } as unknown as AgentKindDeps)
  h.open()
  return { calls, options: h.options as Options, models, unreachable }
}

beforeEach(() => {
  h.options = null
})

describe('the agent conversation', () => {
  it('opens a file at runs through the tab openers', () => {
    const one = setup()

    one.options.openFileAt!('a.md', 'run1', 'run2')

    expect(one.calls).toStrictEqual([['openFileAt', 'a.md', ['run1', 'run2']]])
  })

  it('opens a file beside the conversation', () => {
    const one = setup()

    one.options.openFileBeside!('a.md')

    expect(one.calls).toStrictEqual([['openFile', 'a.md', '', 'beside']])
  })

  it('resolves a written name from the top of the vault', () => {
    const one = setup()

    expect(one.options.resolve!('comp')).toBe('Resolved.md')
    expect(one.calls).toStrictEqual([['resolve', '', 'comp']])
  })

  it('says why the agent is unreachable', () => {
    const one = setup()

    expect(one.options.unreachable!()).toBe('down')
    one.unreachable.value = ''
    expect(one.options.unreachable!()).toBe('')
  })

  it('writes and reads settings through the settings of the window', () => {
    const one = setup()

    expect(one.options.writeSetting!('a.b', 'x')).toBe('written')
    expect(one.options.getSetting!('a.b')).toBe('value')
    expect(one.calls).toStrictEqual([
      ['writeSetting', 'a.b', 'x'],
      ['getSetting', 'a.b'],
    ])
  })

  it('reads the models the settings hold at a path', () => {
    const one = setup()

    expect(one.options.getModelsAt!(['agent', 'use'])).toBe(one.models.value)
  })
})
