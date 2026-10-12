/**
 * The presets of a vault, read by the client the schema generates.
 *
 * A preset names only the settings a person moved, so what is proved here is
 * what the window makes of the rest: a key the file does not carry stands at
 * the default, and a bound the application said nothing about is absent.
 */
import { describe, expect, it, vi } from 'vitest'
import { asFailure } from '@numen/wire'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

/** What the window asked for, as the transport wrote it out. */
let asked: Record<string, unknown>[] = []

/** What the application answers with, in the words the schema writes it in. */
const replyWith = (answer: unknown) => {
  asked = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (_url: string, init: { body: Uint8Array }) => {
      asked.push(JSON.parse(new TextDecoder().decode(init.body)))
      return new Response(JSON.stringify(answer), {
        headers: { 'content-type': 'application/json' },
      })
    }),
  )
}

const { DEFAULTS, NO_CURVE_POINT } = await import('../lib/presets')
const { presets } = await import('./presets.wire')

describe('the settings of a preset', () => {
  it('are the defaults where the file names none of them', async () => {
    replyWith({ preset: { path: 'Daily.md', title: 'Daily' } })

    const answer = await presets.read('Daily.md')
    expect(answer.ok ? answer.value.preset?.settings : null).toEqual(DEFAULTS)
  })

  it('carry the rule and the unit in the words the window uses', async () => {
    replyWith({
      preset: {
        path: 'Daily.md',
        title: 'Daily',
        settings: {
          goal: 'GOAL_RETENTION',
          minutesADay: 30,
          counts: 'BUDGET_UNIT_SHOWS',
          learned: 'RULE_RETENTION',
          load: { mon: 50 },
        },
      },
      at: { path: 'Daily.md', size: '12', mtime: '34' },
    })

    const answer = await presets.read('Daily.md')

    if (!answer.ok) throw new Error('the preset was refused')
    expect(answer.value.preset?.settings).toMatchObject({
      goal: 'retention',
      minutesADay: 30,
      counts: 'shows',
      learned: 'retention',
      load: { mon: 50 },
    })
    expect(answer.value.at).toBe('12 34 Daily.md')
  })

  it('take the default for a rule the file leaves unnamed', async () => {
    replyWith({
      preset: { path: 'Daily.md', title: 'Daily', settings: { learned: 'RULE_UNSPECIFIED' } },
    })

    const answer = await presets.read('Daily.md')
    expect(answer.ok ? answer.value.preset?.settings.learned : null).toBe(DEFAULTS.learned)
  })

  it('are no preset at all where the read was refused', async () => {
    replyWith({ error: 'ERROR_CODE_NOT_A_PRESET' })

    const answer = await presets.read('Notes.md')

    expect(answer).toEqual(asFailure('notAPreset'))
  })
})

describe('how far each setting goes', () => {
  it('carries the ends the application named and nothing else', async () => {
    replyWith({
      preset: { path: 'Daily.md', title: 'Daily' },
      bounds: { minutesADay: { least: 5, most: 240 }, load: { least: 0, most: 100 } },
    })

    const answer = await presets.read('Daily.md')
    expect(answer.ok ? answer.value.bounds : null).toEqual({
      minutesADay: { least: 5, most: 240 },
      load: { least: 0, most: 100 },
    })
  })

  it('is nothing at all until the application has said', async () => {
    replyWith({ preset: { path: 'Daily.md', title: 'Daily' } })

    const answer = await presets.read('Daily.md')
    expect(answer.ok ? answer.value.bounds : null).toEqual({})
  })
})

describe('the presets of a vault', () => {
  it('come back as what each is called and where it stands', async () => {
    replyWith({ presets: [{ path: 'Daily.md', title: 'Daily' }] })

    expect(await presets.list()).toEqual([{ path: 'Daily.md', title: 'Daily' }])
  })
})

describe('making a preset', () => {
  it('answers where it was filed', async () => {
    replyWith({ path: 'Presets/Daily.md' })

    expect(await presets.createPreset('Daily', 'Presets')).toEqual({
      ok: true,
      value: { path: 'Presets/Daily.md' },
    })
  })
})

describe('putting a deck on a preset', () => {
  it('names the file the window read', async () => {
    replyWith({ at: { path: 'Deck.md', size: '12', mtime: '34' } })

    const answer = await presets.scheduleDeck('Deck.md', 'Daily.md', '12 34 Deck.md')

    expect(asked[0]?.baseline).toEqual({ path: 'Deck.md', size: '12', mtime: '34' })
    expect(answer.ok && answer.value.at).toBe('12 34 Deck.md')
  })

  it('names no file where the window read none', async () => {
    replyWith({})

    await presets.scheduleDeck('Deck.md', '', '')

    expect(asked[0]?.baseline).toBeUndefined()
  })

  it('says the file moved past what the window read', async () => {
    replyWith({ error: 'ERROR_CODE_STALE' })

    expect(await presets.scheduleDeck('Deck.md', 'Daily.md', '12 34 Deck.md')).toEqual(
      asFailure('changed'),
    )
  })
})

describe('the preset a deck is scheduled by', () => {
  it('is read under the deck and not under a path', async () => {
    replyWith({ preset: { path: '', title: '' } })

    const answer = await presets.getDeckPreset('Deck.md')
    expect(answer.ok ? answer.value.preset?.path : null).toBe('')
    expect(asked[0]).toEqual({ deck: 'Deck.md' })
  })
})

describe('writing settings into a preset', () => {
  it('sends the rule and the unit as the schema names them', async () => {
    replyWith({ at: { path: 'Daily.md', size: '12', mtime: '34' } })

    await presets.write(
      'Daily.md',
      { ...DEFAULTS, counts: 'shows', learned: 'retention', load: { mon: 50 } },
      '',
    )

    expect(asked[0]?.settings).toMatchObject({
      goal: 'GOAL_MINUTES_A_DAY',
      counts: 'BUDGET_UNIT_SHOWS',
      learned: 'RULE_RETENTION',
      load: { mon: 50 },
    })
  })
})

describe('what a preset comes to over the range of its goal', () => {
  it('carries every place, and the day one is learned on where the rule has one', async () => {
    replyWith({
      curve: {
        goal: 'GOAL_MINUTES_A_DAY',
        grid: [10, 20],
        days: [],
        at: [
          { reviews: 40, minutes: 10, clears: 3, learns: 30, backlog: [4, 2] },
          { reviews: 80, minutes: 20, clears: 1, backlog: [] },
        ],
        now: { at: 0, value: 10, day: '' },
        decks: 2,
        cards: 400,
        overdue: 12,
      },
    })

    const curve = await presets.curve('Daily.md', DEFAULTS)

    expect(curve.at.map((one) => one.learns)).toEqual([30, undefined])
    expect(curve.at[0]?.backlog).toEqual([4, 2])
    expect(curve).toMatchObject({
      goal: 'minutes',
      decks: 2,
      cards: 400,
      overdue: 12,
      isHonest: true,
    })
  })

  it('stands nowhere where the answer suggests no place', async () => {
    replyWith({ curve: { goal: 'GOAL_MINUTES_A_DAY', grid: [], days: [], at: [] } })

    const curve = await presets.curve('Daily.md', DEFAULTS)

    expect(curve.now).toEqual(NO_CURVE_POINT)
    expect(curve.suggested).toEqual(NO_CURVE_POINT)
  })

  it('is an empty one where the answer carries no curve at all', async () => {
    replyWith({})

    expect(await presets.curve('Daily.md', DEFAULTS)).toMatchObject({
      goal: DEFAULTS.goal,
      grid: [],
      at: [],
      decks: 0,
    })
  })
})

describe('what the requests of a preset name', () => {
  it('asks for a preset under its path', async () => {
    replyWith({ preset: { path: 'Daily.md', title: 'Daily' } })

    await presets.read('Daily.md')

    expect(asked[0]).toEqual({ path: 'Daily.md' })
  })

  it('files a new preset by its title in the folder', async () => {
    replyWith({ path: 'Presets/Daily.md' })

    await presets.createPreset('Daily', 'Presets')

    expect(asked[0]).toEqual({ title: 'Daily', path: 'Presets' })
  })

  it('names the deck, the preset and the file the window read when scheduling', async () => {
    replyWith({})

    await presets.scheduleDeck('Deck.md', 'Daily.md', '12 34 Deck.md')

    expect(asked[0]).toMatchObject({ deck: 'Deck.md', preset: 'Daily.md' })
  })

  it('sends the path and the settings of a curve it asks for', async () => {
    replyWith({})

    await presets.curve('Daily.md', { ...DEFAULTS, minutesADay: 45 })

    expect(asked[0]).toMatchObject({ path: 'Daily.md', settings: { minutesADay: 45 } })
  })
})

describe('the stamp of a file an answer is about', () => {
  it('is the empty string where a read answered with none', async () => {
    replyWith({ preset: { path: 'Daily.md', title: 'Daily' } })

    const answer = await presets.read('Daily.md')

    expect(answer.ok ? answer.value.at : null).toBe('')
  })

  it('is the empty string where scheduling answered with none', async () => {
    replyWith({})

    const answer = await presets.scheduleDeck('Deck.md', 'Daily.md', '')

    expect(answer.ok ? answer.value.at : null).toBe('')
  })

  it('is the empty string where a write answered with none', async () => {
    replyWith({})

    const answer = await presets.write('Daily.md', DEFAULTS, '')

    expect(answer.ok ? answer.value.at : null).toBe('')
  })

  it('is the file a write came to', async () => {
    replyWith({ at: { path: 'Daily.md', size: '12', mtime: '34' } })

    const answer = await presets.write('Daily.md', DEFAULTS, '')

    expect(answer).toEqual({ ok: true, value: { at: '12 34 Daily.md' } })
  })
})

describe('writing into a preset the window read', () => {
  it('names the file the window read', async () => {
    replyWith({ at: { path: 'Daily.md', size: '12', mtime: '34' } })

    await presets.write('Daily.md', DEFAULTS, '12 34 Daily.md')

    expect(asked[0]).toMatchObject({
      path: 'Daily.md',
      baseline: { path: 'Daily.md', size: '12', mtime: '34' },
    })
  })

  it('names no file where the window read none', async () => {
    replyWith({})

    await presets.write('Daily.md', DEFAULTS, '')

    expect(asked[0]?.baseline).toBeUndefined()
  })

  it('says the file moved past what the window read', async () => {
    replyWith({ error: 'ERROR_CODE_STALE' })

    expect(await presets.write('Daily.md', DEFAULTS, '12 34 Daily.md')).toEqual(
      asFailure('changed'),
    )
  })

  it('names the refusal in the words the window uses', async () => {
    replyWith({ error: 'ERROR_CODE_NOT_A_PRESET' })

    expect(await presets.write('Notes.md', DEFAULTS, '')).toEqual(asFailure('notAPreset'))
  })

  it('names the refusal of a scheduling in the words the window uses', async () => {
    replyWith({ error: 'ERROR_CODE_NOT_A_PRESET' })

    expect(await presets.scheduleDeck('Deck.md', 'Notes.md', '')).toEqual(asFailure('notAPreset'))
  })
})

describe('the ends of every setting', () => {
  it('are carried for each setting the application named', async () => {
    const ends = { least: 1, most: 9 }
    replyWith({
      preset: { path: 'Daily.md', title: 'Daily' },
      bounds: {
        minutesADay: ends,
        newADay: ends,
        reviewsADay: ends,
        retention: ends,
        backlog: ends,
        interval: ends,
        load: ends,
      },
    })

    const answer = await presets.read('Daily.md')

    expect(answer.ok ? answer.value.bounds : null).toEqual({
      minutesADay: ends,
      newADay: ends,
      reviewsADay: ends,
      retention: ends,
      backlog: ends,
      interval: ends,
      load: ends,
    })
  })

  it('are left out for each setting the application did not name', async () => {
    replyWith({
      preset: { path: 'Daily.md', title: 'Daily' },
      bounds: { interval: { least: 1, most: 9 } },
    })

    const answer = await presets.read('Daily.md')

    expect(answer.ok ? answer.value.bounds : null).toEqual({ interval: { least: 1, most: 9 } })
  })
})

describe('the curve of a goal other than the default', () => {
  it('carries the goal the answer names', async () => {
    replyWith({ curve: { goal: 'GOAL_RETENTION', grid: [], days: [], at: [] } })

    expect((await presets.curve('Daily.md', DEFAULTS)).goal).toBe('retention')
  })

  it('is valid and honest', async () => {
    replyWith({ curve: { goal: 'GOAL_RETENTION', grid: [], days: [], at: [] } })

    expect(await presets.curve('Daily.md', DEFAULTS)).toMatchObject({
      isValid: true,
      isHonest: true,
    })
  })

  it('carries the places the answer names', async () => {
    replyWith({
      curve: {
        goal: 'GOAL_MINUTES_A_DAY',
        grid: [],
        days: [],
        at: [],
        now: { at: 3, value: 25, day: 'mon' },
        suggested: { at: 5, value: 40, day: 'tue' },
      },
    })

    const curve = await presets.curve('Daily.md', DEFAULTS)

    expect(curve.now).toEqual({ at: 3, value: 25, day: 'mon' })
    expect(curve.suggested).toEqual({ at: 5, value: 40, day: 'tue' })
  })

  it('names no learned figure for a place that has none', async () => {
    replyWith({
      curve: {
        goal: 'GOAL_MINUTES_A_DAY',
        grid: [10],
        days: [],
        at: [{ reviews: 1, minutes: 10 }],
      },
    })

    const curve = await presets.curve('Daily.md', DEFAULTS)

    expect('learns' in (curve.at[0] ?? { learns: 0 })).toBe(false)
  })
})
