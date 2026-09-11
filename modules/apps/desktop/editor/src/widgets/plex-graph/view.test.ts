/**
 * What one plex is showing, and what travelling comes to.
 *
 * Every one of these has a failure that looks like nothing at all: a plex
 * showing something stale, with no error and no way back.
 */
import { describe, expect, it } from 'vitest'
import { usePlexView, type Neighbours } from './view'
import type { Neighbourhood } from '../../shared/core'

const answer = (path: string): Neighbourhood => ({
  focus: { path, title: path },
  focusType: 'note',
  related: [],
})

/** A neighbourhood of a note the index does not hold: a focus with no path. */
const nothing = (): Neighbourhood => ({
  focus: { path: '', title: '' },
  focusType: 'note',
  related: [],
})

/** A vault that answers whatever it is told to. */
const fake = (neighbourhood: Neighbours['neighbourhood']): Neighbours => ({ neighbourhood })

describe('two questions in flight', () => {
  it('keeps the answer to the last one asked, however they come back', async () => {
    const delays: Record<string, number> = { Slow: 30, Fast: 0 }
    const plex = usePlexView(
      fake(async (path) => {
        await new Promise((wake) => setTimeout(wake, delays[path] ?? 0))
        return answer(path)
      }),
    )

    const slow = plex.go('Slow')
    const fast = plex.go('Fast')
    await Promise.all([slow, fast])

    expect(plex.here.value).toBe('Fast')
    expect(plex.neighbourhood.value?.focus.path).toBe('Fast')
  })
})

describe('a note that moved while a question was in flight', () => {
  it('lets go of the answer about where it was', async () => {
    let release = () => {}
    let holding: Promise<void> | null = null
    const plex = usePlexView(
      fake(async (path) => {
        if (holding) await holding
        return answer(path)
      }),
    )
    await plex.go('Note.md')

    holding = new Promise<void>((wake) => (release = wake))
    const asking = plex.go('Note.md')
    plex.follows([{ from: 'Note.md', to: 'moved/Note.md' }])
    release()
    await asking

    expect(plex.here.value).toBe('moved/Note.md')
    expect(plex.neighbourhood.value?.focus.path).toBe('Note.md')
  })

  it('holds on to an answer about a note nothing moved', async () => {
    let release = () => {}
    let holding: Promise<void> | null = null
    const plex = usePlexView(
      fake(async (path) => {
        if (holding) await holding
        return answer(path)
      }),
    )

    holding = new Promise<void>((wake) => (release = wake))
    const asking = plex.go('Note.md')
    plex.follows([{ from: 'Elsewhere.md', to: 'moved/Elsewhere.md' }])
    release()
    await asking

    expect(plex.here.value).toBe('Note.md')
  })
})

describe('the note in focus goes away', () => {
  it('says so, keeps what it is showing, and can come back to it', async () => {
    let holds = true
    const plex = usePlexView(fake(async (path) => (holds ? answer(path) : nothing())))

    await plex.go('Note.md')
    expect(plex.here.value).toBe('Note.md')

    holds = false
    await plex.go('Note.md')
    expect(plex.error.value).toContain('Note.md')
    // The path it asked about is kept, which is the only way back to it.
    expect(plex.here.value).toBe('Note.md')
    expect(plex.neighbourhood.value?.focus.path).toBe('Note.md')

    holds = true
    await plex.go('Note.md')
    expect(plex.error.value).toBe('')
  })
})

describe('a plex whose tab has closed', () => {
  it('says nothing about the answer that was on its way', async () => {
    const plex = usePlexView(
      fake(async () => {
        await new Promise((wake) => setTimeout(wake, 0))
        return nothing()
      }),
    )

    const going = plex.go('Gone.md')
    plex.close()
    await going

    expect(plex.error.value).toBe('')
  })

  it('travels nowhere else', async () => {
    const asked: string[] = []
    const plex = usePlexView(
      fake(async (path) => {
        asked.push(path)
        return answer(path)
      }),
    )

    plex.close()
    await plex.go('Note.md')

    expect(asked).toStrictEqual([])
    expect(plex.here.value).toBe('')
  })
})

describe('two plexes', () => {
  it('travel apart from one another', async () => {
    const vault = fake(async (path) => answer(path))
    const one = usePlexView(vault)
    const two = usePlexView(vault)

    await one.go('One.md')
    await two.go('Two.md')

    expect(one.here.value).toBe('One.md')
    expect(two.here.value).toBe('Two.md')
    expect(one.neighbourhood.value?.focus.path).toBe('One.md')
    expect(two.neighbourhood.value?.focus.path).toBe('Two.md')
  })
})

describe('a vault that changed somewhere else', () => {
  it('leaves the picture on screen standing, answer for answer', async () => {
    const plex = usePlexView(fake(async (path) => answer(path)))
    await plex.go('Entropy.md')
    const drawn = plex.neighbourhood.value

    await plex.go('Entropy.md')

    expect(plex.neighbourhood.value).toBe(drawn)
  })

  it('draws again where what is around the note is different', async () => {
    let related: string[] = []
    const plex = usePlexView(
      fake(async (path) => ({
        focus: { path, title: path },
        focusType: 'note',
        related: related.map((to) => ({
          path: to,
          title: to,
          type: 'note' as const,
          seat: 'child' as const,
          label: '',
          through: '',
          mutual: false,
        })),
      })),
    )
    await plex.go('Entropy.md')
    const drawn = plex.neighbourhood.value

    related = ['Heat.md']
    await plex.go('Entropy.md')

    expect(plex.neighbourhood.value).not.toBe(drawn)
    expect(plex.neighbourhood.value?.related).toHaveLength(1)
  })
})
