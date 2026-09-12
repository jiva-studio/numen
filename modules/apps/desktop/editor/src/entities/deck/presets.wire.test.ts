/**
 * The presets of a vault, read by the client the schema generates.
 *
 * A preset names only the settings a person moved, so what is proved here is
 * what the window makes of the rest: a key the file does not carry stands at
 * the default, and a bound the application said nothing about is absent.
 */
import { describe, expect, it, vi } from 'vitest'

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

const { DEFAULTS, NOWHERE } = await import('./presets')
const { presets } = await import('./presets.wire')

describe('the settings of a preset', () => {
  it('are the defaults where the file names none of them', async () => {
    replyWith({ preset: { path: 'Daily.md', title: 'Daily' } })

    expect((await presets.read('Daily.md')).preset?.settings).toEqual(DEFAULTS)
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

    expect(answer.preset?.settings).toMatchObject({
      goal: 'retention',
      minutesADay: 30,
      counts: 'shows',
      learned: 'retention',
      load: { mon: 50 },
    })
    expect(answer.at).toBe('12 34 Daily.md')
  })

  it('take the default for a rule the file leaves unnamed', async () => {
    replyWith({
      preset: { path: 'Daily.md', title: 'Daily', settings: { learned: 'RULE_UNSPECIFIED' } },
    })

    expect((await presets.read('Daily.md')).preset?.settings.learned).toBe(DEFAULTS.learned)
  })

  it('are no preset at all where the read was refused', async () => {
    replyWith({ error: 'ERROR_CODE_NOT_A_PRESET' })

    const answer = await presets.read('Notes.md')

    expect(answer.preset).toBeNull()
    expect(answer.error).toBe('notAPreset')
  })
})

describe('how far each setting goes', () => {
  it('carries the ends the application named and nothing else', async () => {
    replyWith({
      preset: { path: 'Daily.md', title: 'Daily' },
      bounds: { minutesADay: { least: 5, most: 240 }, load: { least: 0, most: 100 } },
    })

    expect((await presets.read('Daily.md')).bounds).toEqual({
      minutesADay: { least: 5, most: 240 },
      load: { least: 0, most: 100 },
    })
  })

  it('is nothing at all until the application has said', async () => {
    replyWith({ preset: { path: 'Daily.md', title: 'Daily' } })

    expect((await presets.read('Daily.md')).bounds).toEqual({})
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
      path: 'Presets/Daily.md',
      error: null,
    })
  })
})

describe('putting a deck on a preset', () => {
  it('names the file the window read', async () => {
    replyWith({ at: { path: 'Deck.md', size: '12', mtime: '34' } })

    const answer = await presets.scheduleDeck('Deck.md', 'Daily.md', '12 34 Deck.md')

    expect(asked[0]?.seen).toEqual({ path: 'Deck.md', size: '12', mtime: '34' })
    expect(answer.at).toBe('12 34 Deck.md')
  })

  it('names no file where the window read none', async () => {
    replyWith({})

    await presets.scheduleDeck('Deck.md', '', '')

    expect(asked[0]?.seen).toBeUndefined()
  })

  it('says the file moved past what the window read', async () => {
    replyWith({ error: 'ERROR_CODE_STALE' })

    expect(await presets.scheduleDeck('Deck.md', 'Daily.md', '12 34 Deck.md')).toEqual({
      error: null,
      changed: true,
      at: '',
    })
  })
})

describe('the preset a deck is scheduled by', () => {
  it('is read under the deck and not under a path', async () => {
    replyWith({ preset: { path: '', title: '' } })

    expect((await presets.scheduling('Deck.md')).preset?.path).toBe('')
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
    expect(curve).toMatchObject({ goal: 'minutes', decks: 2, cards: 400, overdue: 12, honest: true })
  })

  it('stands nowhere where the answer suggests no place', async () => {
    replyWith({ curve: { goal: 'GOAL_MINUTES_A_DAY', grid: [], days: [], at: [] } })

    const curve = await presets.curve('Daily.md', DEFAULTS)

    expect(curve.now).toEqual(NOWHERE)
    expect(curve.suggested).toEqual(NOWHERE)
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
