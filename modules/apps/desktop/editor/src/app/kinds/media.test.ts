/** What the url kind is handed by the window. */
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
    options: { deps: unknown; as: unknown },
  ) => {
    const name = 'pointed'
    h.made.push({ createState, deps: options.deps, as: options.as })
    return { updateTasks: (tasks: unknown) => h.updated.push([name, tasks]) }
  },
}))
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
  const [pointed] = h.made as Made[]
  return { kinds, commands, tasks, pointed: pointed! }
}

beforeEach(() => {
  vi.clearAllMocks()
  h.made.length = 0
  h.updated.length = 0
})

describe('the url kind', () => {
  it('is drawn as its own medium', () => {
    const one = setup()

    expect(one.pointed.as).toStrictEqual({ name: 'urls medium' })
  })

  it('reads a transcript of the recordings at the path, with the player probe', () => {
    const one = setup()

    one.pointed.createState('b.mp3')

    expect(h.useTranscript).toHaveBeenCalledWith({ name: 'recordings' }, 'b.mp3', { plays })
  })

  it('runs a command over the file, as the source of its medium, with what the vault makes of it', () => {
    const one = setup()

    one.pointed.deps.runCommand('proofread', 'none.mp3', 'Another')

    expect(one.commands).toStrictEqual([
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

  it('asks the window whether a run can be made', () => {
    const one = setup()

    expect(one.pointed.deps.canRun('runnable')).toBe(true)
    expect(one.pointed.deps.canRun('other')).toBe(false)
  })

  it('is handed the tasks of the window as they change', async () => {
    const one = setup()
    const next = [{ id: 'task' }]

    one.tasks.value = next
    await Promise.resolve()

    expect(h.updated).toStrictEqual([['pointed', next]])
  })
})
