/**
 * The presets the window has open, and what one preset tab holds.
 *
 * A preset is read whole and written whole: the settings a person moves are one
 * group, and one write carries them. The file a read came out of goes back with
 * that write, so a file that moved under the window is answered and not
 * overwritten.
 */
import { ref, shallowRef, type Ref } from 'vue'
import type { PlexShowing } from '@numen/ui'
import type { Refused, Went } from '../core'
import type { Says } from '../telling'
import type { Host, Kind } from '../windowing'
import type { Putting } from '../putting'
import { PRESET } from '../workspace'
import PresetTab from './PresetTab.vue'
import { DEFAULTS, type Curve, type Goal, type Presets, type Settings } from './core'
import {
  approximate,
  dayAfter,
  daysUntil,
  following,
  held,
  isDay,
  kept,
  nearest,
  offGoal,
  producing,
  shapeOf,
  standing,
  steers,
  type Field,
} from './curve'
import { WORDS as words } from './words'

/** How far off the day a goal of a date opens on, where the file names none. */
const AHEAD = 30

/** What one preset tab holds. */
export interface Held {
  /** The identity this preset opened under, which its tab keeps wherever it goes. */
  readonly id: string
  /** The settings as they now stand, whether or not they have been written. */
  settings(): Settings
  /** The curve of the goal, which is the control the person moves. */
  curve(): Curve
  /** Where the knob stands on that curve. */
  place(): number
  /** The fields a person typed themselves, which no longer follow the goal. */
  byHand(): ReadonlySet<Field>
  /** What is wrong with the file, in the words to show. */
  problems(): readonly string[]
  /** What the file was refused for, in words a person reads, or nothing. */
  saying(): string
  /** The file moved under the window and nothing was written. */
  changed(): boolean
  /** The file read again, which is the way out of that. */
  again(): void
  /** Another of the three goals steers this preset from now on. */
  chooses(goal: Goal): void
  /** The knob moved to a place of the grid. Nothing is written while it moves. */
  moves(place: number): void
  /** The knob let go of, which is what writes the group. */
  settles(): void
  /**
   * One field typed by hand, which keeps what was typed and stops following.
   * The field the goal steers is the knob, and typing into it moves the knob.
   */
  types(field: Field, value: number | string | boolean | readonly string[]): void
  /** That field put back under the goal. */
  follows(field: Field): void
  /** The tab is closing. */
  shuts(id: string): void
}

export function presetting(
  core: Presets,
  host: Host,
  puts: Putting,
  said: Says,
  today: () => Date = () => new Date(),
) {
  /** What each preset is called, as the vault last read it. */
  const titles = new Map<string, string>()

  /** Everything one open preset stands at. */
  interface Kept {
    readonly path: Ref<string>
    readonly settings: Ref<Settings>
    readonly curve: Ref<Curve>
    readonly place: Ref<number>
    readonly problems: Ref<readonly string[]>
    readonly changed: Ref<boolean>
    readonly saying: Ref<string>
    /** The file the settings came out of, presented at the next write. */
    at: string
    /** Which curve is the current one. An answer for a goal since left is dropped. */
    asked: number
    /** The settings the curve in hand was asked under, as `shapeOf` reads them. */
    shape: string
  }

  const open = new Map<string, Kept>()

  const keeps = (path: string): Kept => ({
    path: ref(path),
    settings: shallowRef<Settings>(DEFAULTS),
    curve: shallowRef<Curve>(approximate(DEFAULTS, today())),
    place: ref(0),
    problems: shallowRef<readonly string[]>([]),
    changed: ref(false),
    saying: ref(''),
    at: '',
    asked: 0,
    shape: '',
  })

  /** What one refusal is put in, and the window's own word for the rest. */
  const whyOf = (refusal: Refused | null): string => {
    if (refusal === null) return ''
    if (refusal === 'notAPreset') return words.notAPreset
    return words.refused
  }

  /** The settings of one preset, read again from the file. */
  const reads = async (one: Kept): Promise<void> => {
    let answer
    try {
      answer = await core.read(one.path.value)
    } catch (error) {
      console.error(error)
      one.saying.value = words.unreachable
      return
    }
    one.saying.value = whyOf(answer.refusal)
    one.changed.value = false
    one.at = answer.at
    if (!answer.preset) return
    if (answer.preset.title) titles.set(one.path.value, answer.preset.title)
    one.problems.value = answer.preset.problems
    one.settings.value = answer.preset.settings
    if (shapeOf(one.settings.value) !== one.shape) await curves(one)
  }

  /**
   * The curve of the goal these settings name. The line the window works out
   * for itself stands in its place until the application answers. Where a
   * curve of this goal already stands, the knob stays where it is and only the
   * line is drawn again.
   */
  const curves = async (one: Kept): Promise<void> => {
    const mine = ++one.asked
    one.shape = shapeOf(one.settings.value)
    const standsAlready = one.curve.value.honest && one.curve.value.goal === one.settings.value.goal
    if (!standsAlready) {
      const guess = approximate(one.settings.value, today())
      one.curve.value = guess
      one.place.value = Math.max(guess.now.at, 0)
    }
    let answer: Curve
    try {
      answer = await core.curve(one.path.value, one.settings.value)
    } catch (error) {
      // The line the window guessed stands, and it says that it is a guess.
      console.error(error)
      return
    }
    if (mine !== one.asked) return
    one.curve.value = answer
    if (standsAlready) return
    one.place.value =
      answer.now.at >= 0
        ? answer.now.at
        : Math.max(nearest(answer.grid, standing(one.settings.value, today())), 0)
  }

  /** The settings as they now stand, written into the file the read came out of. */
  const writes = async (one: Kept): Promise<void> => {
    said('')
    let answer
    try {
      answer = await core.write(one.path.value, one.settings.value, one.at)
    } catch (error) {
      one.saying.value = words.unreachable
      console.error(error)
      return
    }
    if (answer.changed) {
      one.changed.value = true
      return
    }
    if (answer.refusal) {
      one.saying.value = answer.refusal === 'notAPreset' ? words.notAPreset : words.notSaved
      said(words.notSaved, 'refusal')
      return
    }
    one.saying.value = ''
    one.at = answer.at
  }

  /** The settings the place the knob stands at produces, with what was typed kept. */
  const turns = (one: Kept, place: number): void => {
    const off = offGoal(one.settings.value, one.curve.value, today())
    const produced = producing(one.settings.value, place, one.curve.value, today())
    one.settings.value = kept(produced, one.settings.value, off)
    one.place.value = place
  }

  /** One field of the settings, as a person typed it. */
  const typed = (
    settings: Settings,
    field: Field,
    value: number | string | boolean | readonly string[],
  ): Settings => {
    if (field === 'byDate' && typeof value === 'string') return { ...settings, byDate: value }
    if (field === 'counts' && (value === 'cards' || value === 'shows')) {
      return { ...settings, counts: value }
    }
    if (field === 'lightDays' && Array.isArray(value)) return { ...settings, lightDays: value }
    if (field === 'evenLoad' && typeof value === 'boolean') return { ...settings, evenLoad: value }
    if (typeof value !== 'number') return settings
    if (field === 'retention') return { ...settings, retention: held(value, 'retention') }
    if (field === 'newADay') return { ...settings, newADay: held(value, 'newADay') }
    if (field === 'reviewsADay') return { ...settings, reviewsADay: held(value, 'reviewsADay') }
    if (field === 'minutesADay') return { ...settings, minutesADay: held(value, 'minutesADay') }
    return settings
  }

  /** Where a value typed into the field the goal steers falls on the grid. */
  const falling = (one: Kept, value: number | string | boolean | readonly string[]): number => {
    if (typeof value === 'number') return nearest(one.curve.value.grid, value)
    if (typeof value === 'string' && isDay(value)) {
      return nearest(one.curve.value.grid, daysUntil(today(), value))
    }
    return -1
  }

  /** A goal of a date opens on a day, so one is named where the file names none. */
  const aiming = (settings: Settings, goal: Goal): Settings =>
    goal === 'date' && (settings.byDate === '' || daysUntil(today(), settings.byDate) <= 0)
      ? { ...settings, goal, byDate: dayAfter(today(), AHEAD) }
      : { ...settings, goal }

  const holds = (id: string): Held => {
    const one = open.get(id) ?? keeps(id)
    return {
      id,
      settings: () => one.settings.value,
      curve: () => one.curve.value,
      place: () => one.place.value,
      byHand: () => offGoal(one.settings.value, one.curve.value, today()),
      problems: () => one.problems.value,
      saying: () => one.saying.value,
      changed: () => one.changed.value,
      again: () => void reads(one),
      chooses: (goal) => {
        if (goal === one.settings.value.goal) return
        one.settings.value = aiming(one.settings.value, goal)
        void curves(one).then(() => writes(one))
      },
      moves: (place) => turns(one, place),
      settles: () => void writes(one),
      types: (field, value) => {
        // What is typed into the field the goal steers moves the knob there, so
        // the knob, the field and the file stand at one value.
        if (field === steers(one.settings.value.goal)) {
          one.settings.value = typed(one.settings.value, field, value)
          const place = falling(one, value)
          if (place >= 0) turns(one, place)
          void writes(one)
          return
        }
        one.settings.value = typed(one.settings.value, field, value)
        // A field the knob does not ride gives the curve its shape, so the
        // curve is asked for again where one of those is typed.
        if (shapeOf(one.settings.value) !== one.shape) void curves(one)
        void writes(one)
      },
      follows: (field) => {
        one.settings.value = following(one.settings.value, field, one.curve.value, today())
        void writes(one)
      },
      shuts: (tab) => {
        open.delete(id)
        host.closes(tab)
      },
    }
  }

  /** What a preset tab is called: the title the file carries, or the file itself. */
  const called = (path: string): string =>
    titles.get(path) || (path.split('/').pop() ?? path) || words.newPreset

  /**
   * A preset tab as the window keeps it. A preset is its own tab, under the
   * file it stands at, so the same preset asked for twice is the tab it has.
   */
  const kind: Kind<Held> = {
    kind: PRESET,
    opens: (path) => {
      const one = keeps(path)
      open.set(path, one)
      void reads(one)
      return holds(path)
    },
    called: (one) => called(one.id),
    draws: PresetTab,
    identity: (path) => path,
    shuts: (one, id) => {
      one.shuts(id)
      return false
    },
    gone: () => {},
  }

  /** A preset put in front of the person, in a tab of its own. */
  const shows = (path: string, title = '', showing: PlexShowing = 'here'): void => {
    if (title) titles.set(path, title)
    void (showing === 'beside' ? host.beside(PRESET, path) : host.opens(PRESET, path))
  }

  // The editor of a preset, which is its one control and the settings under it.
  // A setting stands on no line of prose, so a preset opens whole.
  puts.holds('preset', shows)

  /** The vault changed: every open preset at one of those paths is read again. */
  const changed = (paths: readonly string[], renamed: readonly Went[] = []): void => {
    for (const went of renamed) {
      const title = titles.get(went.from)
      if (title !== undefined) titles.set(went.to, title)
      titles.delete(went.from)
      const one = open.get(went.from)
      if (!one) continue
      open.delete(went.from)
      one.path.value = went.to
      open.set(went.to, one)
    }
    for (const path of paths) {
      const one = open.get(path)
      if (one) void reads(one)
    }
  }

  return { kind, holds, changed, called, shows }
}
