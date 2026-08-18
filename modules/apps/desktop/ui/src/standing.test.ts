/**
 * Where one plex is standing, and what travelling comes to.
 *
 * Every one of these has a failure that looks like nothing at all: a plex
 * showing something stale, with no error and no way back.
 */
import { describe, expect, it } from 'vitest'
import { standing, type Neighbours } from './standing'
import type { Neighbourhood } from './plex'

const answer = (path: string): Neighbourhood =>
  ({ focus: { path, title: path, identifier: '' }, related: [] }) as unknown as Neighbourhood

/** A neighbourhood of a note the index does not hold: a focus with no path. */
const nothing = () =>
  ({ focus: { path: '', title: '', identifier: '' }, related: [] }) as unknown as Neighbourhood

/** A vault that answers whatever it is told to. */
const fake = (neighbourhood: Neighbours['neighbourhood']): Neighbours => ({ neighbourhood })

describe('two questions in flight', () => {
  it('keeps the answer to the last one asked, however they come back', async () => {
    const delays: Record<string, number> = { Slow: 30, Fast: 0 }
    const plex = standing(
      fake(async (path) => {
        await new Promise((wake) => setTimeout(wake, delays[path] ?? 0))
        return answer(path)
      }),
    )

    const slow = plex.go('Slow')
    const fast = plex.go('Fast')
    await Promise.all([slow, fast])

    expect(plex.here.value).toBe('Fast')
    expect(plex.neighbourhood.value?.focus?.path).toBe('Fast')
  })
})

describe('the note in focus goes away', () => {
  it('says so, keeps what it is showing, and can come back to it', async () => {
    let holds = true
    const plex = standing(fake(async (path) => (holds ? answer(path) : nothing())))

    await plex.go('Note.md')
    expect(plex.here.value).toBe('Note.md')

    holds = false
    await plex.go('Note.md')
    expect(plex.trouble.value).toContain('Note.md')
    // The path it asked about is kept, which is the only way back to it.
    expect(plex.here.value).toBe('Note.md')
    expect(plex.neighbourhood.value?.focus?.path).toBe('Note.md')

    holds = true
    await plex.go('Note.md')
    expect(plex.trouble.value).toBe('')
  })
})

describe('a plex whose tab has closed', () => {
  it('says nothing about the answer that was on its way', async () => {
    const plex = standing(
      fake(async () => {
        await new Promise((wake) => setTimeout(wake, 0))
        return nothing()
      }),
    )

    const going = plex.go('Gone.md')
    plex.close()
    await going

    expect(plex.trouble.value).toBe('')
  })

  it('travels nowhere else', async () => {
    const asked: string[] = []
    const plex = standing(
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
    const one = standing(vault)
    const two = standing(vault)

    await one.go('One.md')
    await two.go('Two.md')

    expect(one.here.value).toBe('One.md')
    expect(two.here.value).toBe('Two.md')
    expect(one.neighbourhood.value?.focus?.path).toBe('One.md')
    expect(two.neighbourhood.value?.focus?.path).toBe('Two.md')
  })
})
