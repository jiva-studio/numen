/** What the recording and url kinds are handed by the window. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createMediaKinds, type MediaKindsDeps } from './media'

interface Made {
  readonly createState: (path: string) => unknown
  readonly deps: {
    runCommand: (id: string, path: string, called: string) => void
    canRun: (run: unknown) => boolean
  }
  readonly as: unknown
}

const h = vi.hoisted(() => ({
  made: [] as unknown[],
  useTranscript: vi.fn((...args: unknown[]) => ({ transcript: args })),
  updated: [] as [string, unknown][],
}))

vi.mock('@/entities/media', () => ({
  recordings: { name: 'recordings' },
  useTranscript: h.useTranscript,
  createRecordingKind: (
    _handle: unknown,
    createState: unknown,
    deps: unknown,
    _openers: unknown,
    as: unknown,
  ) => {
    const name = h.made.length === 0 ? 'recorded' : 'pointed'
    h.made.push({ createState, deps, as })
    return { updateTasks: (tasks: unknown) => h.updated.push([name, tasks]) }
  },
}))
vi.mock('@/pages/media-recording', () => ({ RECORDINGS: { name: 'recordings medium' } }))
vi.mock('@/pages/media-url', () => ({ URLS: { name: 'urls medium' } }))

const plays = { name: 'plays' }

const setup = () => {
  const commands: [string, unknown][] = []
  const tasks = ref<unknown[]>([])
  const canRun = vi.fn((run: unknown) => run === 'runnable')
  const kinds = createMediaKinds({
    tabOpeners: {},
    held: { handle: {} },
    runs: { canRun },
    plays,
    vaults: { makes: ref(new Map([['a.mp3', { flashcards: true }]])) },
    window: { tasks },
    getTarget: () => ({ tab: 't1', vault: 'v' }),
    runCommand: (id: string, at: unknown) => commands.push([id, at]),
  } as unknown as MediaKindsDeps)
  const [recorded, pointed] = h.made as Made[]
  return { kinds, commands, tasks, recorded: recorded!, pointed: pointed! }
}

beforeEach(() => {
  vi.clearAllMocks()
  h.made.length = 0
  h.updated.length = 0
})

describe('the recording and url kinds', () => {
  it('are drawn as their own mediums', () => {
    const one = setup()

    expect(one.recorded.as).toStrictEqual({ name: 'recordings medium' })
    expect(one.pointed.as).toStrictEqual({ name: 'urls medium' })
  })

  it('read a transcript of the recordings at the path, with the player probe', () => {
    const one = setup()

    one.recorded.createState('a.mp3')
    one.pointed.createState('b.mp3')

    expect(h.useTranscript).toHaveBeenNthCalledWith(1, { name: 'recordings' }, 'a.mp3', { plays })
    expect(h.useTranscript).toHaveBeenNthCalledWith(2, { name: 'recordings' }, 'b.mp3', { plays })
  })

  it('run a command over the file, as the source of their medium, with what the vault makes of it', () => {
    const one = setup()

    one.recorded.deps.runCommand('proofread', 'a.mp3', 'A talk')
    one.pointed.deps.runCommand('proofread', 'none.mp3', 'Another')

    expect(one.commands).toStrictEqual([
      [
        'proofread',
        {
          tab: 't1',
          vault: 'v',
          path: '',
          title: 'A talk',
          file: 'a.mp3',
          source: 'recording',
          made: { flashcards: true },
        },
      ],
      [
        'proofread',
        {
          tab: 't1',
          vault: 'v',
          path: '',
          title: 'Another',
          file: 'none.mp3',
          source: 'url',
          made: {},
        },
      ],
    ])
  })

  it('ask the window whether a run can be made', () => {
    const one = setup()

    expect(one.recorded.deps.canRun('runnable')).toBe(true)
    expect(one.pointed.deps.canRun('other')).toBe(false)
  })

  it('are handed the tasks of the window as they change', async () => {
    const one = setup()
    const next = [{ id: 'task' }]

    one.tasks.value = next
    await Promise.resolve()

    expect(h.updated).toStrictEqual([
      ['recorded', next],
      ['pointed', next],
    ])
  })
})
