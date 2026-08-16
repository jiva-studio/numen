/**
 * The rules that decide whether the window keeps up with the vault.
 *
 * Every one of these has a failure that looks like nothing at all: a window
 * showing something stale, with no error and no way back.
 */
import { describe, expect, it } from 'vitest'
import { showing, type Core } from './showing'
import type { Neighbourhood } from './plex'

const answer = (path: string): Neighbourhood =>
  ({ focus: { path, title: path, identifier: '' }, related: [] }) as unknown as Neighbourhood

/** A neighbourhood of a note the index does not hold: a focus with no path. */
const nothing = () =>
  ({ focus: { path: '', title: '', identifier: '' }, related: [] }) as unknown as Neighbourhood

const settled = { name: 'Vault', ready: true, failed: '', unwatched: '' }

/** A core that answers whatever it is told to, and records what it was asked. */
function fake(over: Partial<Core> = {}): Core & { asked: string[] } {
  const asked: string[] = []
  return {
    asked,
    neighbourhood: async (path) => {
      asked.push(path)
      return answer(path)
    },
    opening: async () => ({ path: 'Opening.md' }),
    state: async () => settled,
    // eslint-disable-next-line require-yield
    changes: async function* () {},
    // eslint-disable-next-line require-yield
    focus: async function* () {},
    ...over,
  }
}

const nap = () => new Promise((wake) => setTimeout(wake, 0))

describe('two questions in flight', () => {
  it('keeps the answer to the last one asked, however they come back', async () => {
    const delays: Record<string, number> = { Slow: 30, Fast: 0 }
    const core = fake({
      neighbourhood: async (path) => {
        await new Promise((wake) => setTimeout(wake, delays[path] ?? 0))
        return answer(path)
      },
    })
    const window = showing(core)

    const slow = window.go('Slow')
    const fast = window.go('Fast')
    await Promise.all([slow, fast])

    expect(window.here.value).toBe('Fast')
    expect(window.neighbourhood.value?.focus?.path).toBe('Fast')
  })
})

describe('the note in focus goes away', () => {
  it('says so, keeps what it is showing, and can come back to it', async () => {
    let holds = true
    const core = fake({
      neighbourhood: async (path) => (holds ? answer(path) : nothing()),
    })
    const window = showing(core)

    await window.go('Note.md')
    expect(window.here.value).toBe('Note.md')

    holds = false
    await window.go('Note.md')
    expect(window.notice.value).toContain('Note.md')
    // The path it asked about is kept, which is the only way back to it.
    expect(window.here.value).toBe('Note.md')
    expect(window.neighbourhood.value?.focus?.path).toBe('Note.md')

    holds = true
    await window.go('Note.md')
    expect(window.notice.value).toBe('')
  })
})

describe('the stream of changes', () => {
  it('is taken up again when it ends', async () => {
    let streams = 0
    const core = fake({
      changes: async function* () {
        streams++
        yield { paths: ['Note.md'], reload: false }
      },
    })
    const waits: number[] = []
    const window = showing(core, async (ms) => {
      waits.push(ms)
      if (streams >= 3) window.close()
    })

    await window.follow()

    expect(streams).toBeGreaterThanOrEqual(3)
    expect(waits.length).toBeGreaterThanOrEqual(2)
  })

  it('is taken up again when it fails, and says what happened', async () => {
    let streams = 0
    const core = fake({
      changes: function* () {
        streams++
        throw new Error('connection lost')
      } as unknown as Core['changes'],
    })
    const window = showing(core, async () => {
      if (streams >= 2) window.close()
    })

    await window.follow()

    expect(streams).toBeGreaterThanOrEqual(2)
    expect(window.notice.value).toContain('connection lost')
  })

  it('asks for the picture again for a change to any note', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: ['Somewhere/Else.md'], reload: false }
      },
    })
    const window = showing(core, async () => window.close())

    await window.go('Here.md')
    core.asked.length = 0
    await window.follow()

    expect(core.asked).toContain('Here.md')
  })
})

describe('a vault that could not be read', () => {
  it('stops the waiting and is not called empty', async () => {
    const core = fake({
      opening: async () => null,
      state: async () => ({ ...settled, ready: false, failed: 'permission denied' }),
    })
    const window = showing(core, async () => window.close())

    await window.start()
    await nap()

    expect(window.indexing.value).toBe(false)
    expect(window.trouble.value).toBe('permission denied')
  })
})

describe('a vault that is not being followed', () => {
  it('says so rather than looking up to date', async () => {
    const core = fake({ state: async () => ({ ...settled, unwatched: 'too many watches' }) })
    const window = showing(core, async () => window.close())

    await window.start()
    await nap()

    expect(window.unwatched.value).toBe('too many watches')
  })
})
