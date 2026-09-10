/**
 * Window registration and tab state for flashcard preset tabs.
 */
import { ref, shallowRef, type Ref } from 'vue'
import { StopReason } from '@numen/protocol'
import { answerGuard, type AnswerGuard } from '../shared/questions'
import type { PlexShowing } from '@numen/ui'
import type { Move, RefusalReason } from '../shared/core'
import type { MessageWriter } from '../shared/notices/messages'
import type { TabKind, WindowHandle } from '../shared/tabs/windowTabs'
import type { FileOpeners } from '../shared/tabs/openers'
import { PRESET } from '../shared/tabs/workspace'
import PresetTab from './PresetTab.vue'
import {
  DEFAULTS,
  NO_BOUNDS,
  type Bounds,
  type Curve,
  type Goal,
  type Load,
  type PresetCounts,
  type Presets,
  type Settings,
  type SettingsBounds,
} from './core'
import { dayAfter, daysBetween, isDay } from '@numen/ui'
import {
  approximate,
  goalValue,
  held,
  nearest,
  producing,
  shapeOf,
  steers,
  type Field,
} from './curve'
import { fileOf } from '../shared/paths'
import { WORDS as words } from './words'
import type { PresetTabState, SettingValue } from './types'

export type { PresetTabState, SettingValue }

/** How far off the day a goal of a date opens on, where the file names none. */
const AHEAD = 30

/** Whether what was said is a share for each day of the week that carries one. */
const isLoad = (value: SettingValue): value is Load =>
  typeof value === 'object' && Object.values(value).every((share) => typeof share === 'number')

/**
 * Every day of the week held inside the share one may carry, as the application
 * answered it. A share it has said no bound for is held to none, as every other
 * setting is.
 */
const shares = (load: Load, within: Bounds | undefined): Load => {
  if (within === undefined) return load
  const out: Record<string, number> = {}
  for (const [day, share] of Object.entries(load)) out[day] = held(Math.round(share), within)
  return out
}

/** The settings as a tab holds them while a person is moving them. */
type MutableSettings = { -readonly [field in keyof Settings]: Settings[field] }

/** One setting left where it stands, rather than taken from the read. */
const kept = <F extends keyof Settings>(out: MutableSettings, was: Settings, field: F): void => {
  out[field] = was[field]
}

/**
 * The settings read, under the ones this person has moved and not written. A
 * setting nobody touched is the file's, however many of its neighbours were.
 */
const taking = (read: Settings, was: Settings, theirs: ReadonlySet<keyof Settings>): Settings => {
  if (theirs.size === 0) return read
  const out: MutableSettings = { ...read }
  for (const field of theirs) kept(out, was, field)
  return out
}

/**
 * The presets one window has open. `today` is the review day the window was
 * told: a day of review begins hours past midnight, so a date counted from the
 * calendar day is a day off the count the core answers with.
 */
export function usePresetTab(
  core: Presets,
  handle: WindowHandle,
  puts: FileOpeners,
  said: MessageWriter,
  today: () => string,
) {
  /** What each preset is called, as the vault last read it. */
  const titles = new Map<string, string>()

  /**
   * How far each setting goes. Every read answers the same ones, so they are
   * the window's and not a tab's, and they stand until the first read lands.
   */
  const bounds = shallowRef<SettingsBounds>(NO_BOUNDS)

  /** Everything one open preset stands at. */
  interface OpenPreset {
    readonly path: Ref<string>
    readonly settings: Ref<Settings>
    readonly curve: Ref<Curve>
    /** What the last answer counted the material at, and nothing until one has. */
    readonly material: Ref<PresetCounts | null>
    readonly place: Ref<number>
    readonly problems: Ref<readonly string[]>
    /** Why it schedules nothing on the day the last read was answered in. */
    readonly stopped: Ref<StopReason>
    readonly changed: Ref<boolean>
    readonly saying: Ref<string>
    /** An answer to the picture is on its way, and the tab says it is reading. */
    readonly waiting: Ref<boolean>
    /** The file the settings came out of, presented at the next write. */
    at: string
    /** A write is out, and whether another is wanted once it lands. */
    isWriting: boolean
    writing: boolean
    isSelected: boolean
    wanted: boolean
    /** What the writes now in the air answer to, and null while none is out. */
    flight: Promise<void> | null
    /** The tab was held once at its close and says why; asked again it goes. */
    hasMessage: boolean
    told: boolean
    /** An answer for a goal this tab has since left is dropped. */
    readonly asks: AnswerGuard
    /** A curve is out, and whether the settings moved again while it was. */
    isDrawing: boolean
    drawing: boolean
    shouldDrawAgain: boolean
    drawAgain: boolean
    /** The settings the curve in hand was asked under, as `shapeOf` reads them. */
    shape: string
    /** Whether the curve on screen is an answer, though a newer one may be out. */
    isReal: boolean
    real: boolean
    /**
     * The settings this person has moved, which the file has not been told of.
     * Each is theirs alone: moving one says nothing about the rest.
     */
    readonly theirs: Set<keyof Settings>
    /** Every curve this tab has been answered, under the settings it was asked for. */
    readonly answers: Map<string, Curve>
  }

  const open = new Map<string, OpenPreset>()

  const keeps = (path: string): OpenPreset => ({
    path: ref(path),
    settings: shallowRef<Settings>(DEFAULTS),
    curve: shallowRef<Curve>(approximate(DEFAULTS, today(), bounds.value)),
    material: shallowRef<PresetCounts | null>(null),
    place: ref(0),
    problems: shallowRef<readonly string[]>([]),
    stopped: ref(StopReason.NOTHING),
    changed: ref(false),
    saying: ref(''),
    // A tab opens by reading the file the picture is worked out over.
    waiting: ref(true),
    at: '',
    isWriting: false,
    writing: false,
    isSelected: false,
    wanted: false,
    flight: null,
    hasMessage: false,
    told: false,
    asks: answerGuard(),
    isDrawing: false,
    drawing: false,
    shouldDrawAgain: false,
    drawAgain: false,
    shape: '',
    isReal: false,
    real: false,
    theirs: new Set<keyof Settings>(),
    answers: new Map<string, Curve>(),
  })

  /** What a read was refused for, in words a person reads. */
  const whyOf = (refusal: RefusalReason | null): string =>
    refusal === null ? '' : words.refused(refusal)

  /** The settings of one preset, read again from the file. */
  const reads = async (one: OpenPreset): Promise<void> => {
    let answer
    try {
      answer = await core.read(one.path.value)
    } catch (error) {
      console.error(error)
      one.saying.value = words.unreachable
      one.waiting.value = false
      return
    }
    one.saying.value = whyOf(answer.refusal)
    one.changed.value = false
    one.at = answer.at
    bounds.value = answer.bounds
    // Every curve in hand was worked out over a vault this read has just been
    // through, so each of them is an answer about a vault as it was.
    one.answers.clear()
    if (!answer.preset) {
      one.problems.value = []
      one.stopped.value = StopReason.NOTHING
      one.waiting.value = false
      return
    }
    if (answer.preset.title) titles.set(one.path.value, answer.preset.title)
    one.problems.value = answer.preset.problems
    one.stopped.value = answer.preset.stopsOn
    // A setting a person has moved and not yet written is theirs, and every
    // other one is taken up from the file. Moving one field is not a claim on
    // the eleven beside it, which the tab is standing at a placeholder for
    // until this read lands.
    one.settings.value = taking(answer.preset.settings, one.settings.value, one.theirs)
    await curves(one)
  }

  /**
   * The curve of the goal these settings name. A curve already answered under
   * them is drawn again as it was, so moving between the three goals asks
   * nothing and waits for nothing. Where a curve of this goal already stands
   * over the same range, the knob stays where it is and only the line is drawn
   * again; a range that came back another length, or another set of values, is
   * a grid the old place means nothing on, and the knob goes back to where the
   * preset stands.
   */
  const curves = async (one: OpenPreset): Promise<void> => {
    // A curve already answered stands at once, and a control dragged over a
    // range it has been over reads it without asking again.
    const shape = shapeOf(one.settings.value)
    const held = one.answers.get(shape)
    if (held) {
      one.asks.drop()
      one.shape = shape
      lands(one, held)
      one.place.value = standsAt(held, one.settings.value)
      return
    }
    // One curve is in the air at a time. A hand still moving asks for the
    // settings it comes to rest on, and the range in between is not drawn.
    if (one.isDrawing) {
      one.shouldDrawAgain = true
      one.drawAgain = true
      return
    }
    one.isDrawing = true
    one.drawing = true
    try {
      await drawing(one)
    } finally {
      one.isDrawing = false
      one.drawing = false
    }
    if (!one.shouldDrawAgain) return
    one.shouldDrawAgain = false
    one.drawAgain = false
    await curves(one)
  }

  /** One curve, asked for and landed. */
  const drawing = async (one: OpenPreset): Promise<void> => {
    const mine = one.asks.ask()
    const shape = shapeOf(one.settings.value)
    one.shape = shape
    const riding = one.curve.value.grid

    const standsAlready = (one.isReal ?? one.real) && one.curve.value.goal === one.settings.value.goal
    if (standsAlready) {
      // The numbers on screen were worked out for settings that no longer
      // stand, so they are drawn as the waiting they are: the range and the
      // knob stay where they are and nothing jumps.
      one.curve.value = { ...one.curve.value, honest: false }
    } else {
      // What stands until the answer lands carries the goal and its range, so
      // the readout has units to speak in. It is nobody's answer, so the
      // picture draws no line: the honest one appears once and nothing jumps.
      const meanwhile = approximate(one.settings.value, today(), bounds.value)
      one.curve.value = meanwhile
      one.place.value = Math.max(meanwhile.now.at, 0)
    }
    one.isReal = false
    one.real = false
    one.waiting.value = true
    let answer: Curve
    // A picture nobody is going to answer is not a picture being read: the tab
    // says what happened and stops waiting.
    try {
      answer = await core.curve(one.path.value, one.settings.value)
    } catch (error) {
      console.error(error)
      if (!mine.current) return
      one.saying.value = words.noCurve
      one.waiting.value = false
      return
    }
    if (!mine.current) return
    one.answers.set(shape, answer)
    lands(one, answer)
    if (standsAlready && alike(riding, answer.grid)) return
    one.place.value = standsAt(answer, one.settings.value)
  }

  /**
   * A curve landed and drawn. The figures it counts the material at are kept
   * beside it, so what no setting moves stands on while the next curve is out.
   */
  const lands = (one: OpenPreset, curve: Curve): void => {
    one.curve.value = curve
    one.material.value = {
      decks: curve.decks,
      cards: curve.cards,
      overdue: curve.overdue,
      unbegun: curve.unbegun,
    }
    one.isReal = true
    one.real = true
    one.waiting.value = false
  }

  /** Whether two curves are drawn over the same range, place for place. */
  const alike = (one: readonly number[], two: readonly number[]): boolean =>
    one.length === two.length && one.every((value, at) => value === two[at])

  /** Where the knob stands on a curve: the preset's own place, or the nearest. */
  const standsAt = (curve: Curve, settings: Settings): number =>
    curve.now.at >= 0
      ? curve.now.at
      : Math.max(nearest(curve.grid, goalValue(settings, today())), 0)

  /** The settings as they now stand, into the file the read came out of. */
  const sends = async (one: OpenPreset): Promise<void> => {
    said('')
    let answer
    try {
      answer = await core.write(one.path.value, one.settings.value, one.at)
    } catch (error) {
      one.saying.value = words.unwritten
      console.error(error)
      return
    }
    if (answer.changed) {
      one.changed.value = true
      return
    }
    if (answer.refusal) {
      one.saying.value = words.notSaved(answer.refusal)
      said(one.saying.value, 'refusal')
      return
    }
    one.saying.value = ''
    one.at = answer.at
    // The file has been told of every one of them, so none is still theirs.
    one.theirs.clear()
    one.hasMessage = false
    one.told = false
  }

  /**
   * The settings written. One write per preset is out at a time, and a write
   * asked for while one is out is taken up when that one lands, carrying the
   * file it produced. A file that moved under the window is answered by the
   * person, so nothing is sent on top of that notice.
   */
  const writes = (one: OpenPreset): Promise<void> => {
    if (one.isWriting || one.writing) {
      one.isSelected = true
      one.wanted = true
      return one.flight ?? Promise.resolve()
    }
    one.isWriting = true
    one.writing = true
    const flight = sending(one)
    one.flight = flight
    return flight
  }

  /** One write, and the write asked for while it was out, as one answer. */
  const sending = async (one: OpenPreset): Promise<void> => {
    await sends(one)
    one.isWriting = false
    one.writing = false
    const again = (one.isSelected || one.wanted) && !one.changed.value
    one.isSelected = false
    one.wanted = false
    if (again) {
      await writes(one)
      return
    }
    one.flight = null
  }

  /** Everything the tab owes the file, written and landed. */
  const owed = async (one: OpenPreset): Promise<void> => {
    if (one.theirs.size > 0 && !one.changed.value) await writes(one)
    else if (one.flight) await one.flight
  }

  /** Every open preset writes what it owes, for a window that is going. */
  const flush = async (): Promise<void> => {
    await Promise.all([...open.values()].map(owed))
  }

  /**
   * A tab closing writes what stands unwritten and goes once it lands. A tab
   * whose settings the file would not take is held once and says why; asked a
   * second time it goes, since the person has been told.
   */
  const shut = async (one: OpenPreset): Promise<boolean> => {
    await owed(one)
    if (one.theirs.size === 0 && !one.changed.value) return true
    if (one.hasMessage || one.told) return true
    one.hasMessage = true
    one.told = true
    return false
  }

  /** The settings the place the knob stands at produces, which is its own value. */
  const turns = (one: OpenPreset, place: number): void => {
    const was = one.settings.value
    one.settings.value = producing(was, place, one.curve.value, today(), bounds.value)
    one.place.value = place
    // The knob moves the one value its curve's goal names, and nothing else.
    one.theirs.add(steers(one.curve.value.goal))
  }

  /** One field of the settings, as a person typed it. */
  const typed = (
    settings: Settings,
    field: Field,
    value: SettingValue,
  ): Settings => {
    if (field === 'byDate' && typeof value === 'string') return { ...settings, byDate: value }
    if (field === 'counts' && (value === 'cards' || value === 'shows')) {
      return { ...settings, counts: value }
    }
    if (field === 'learned' && (value === 'interval' || value === 'retention')) {
      return { ...settings, learned: value }
    }
    if (field === 'load' && isLoad(value)) {
      return { ...settings, load: shares(value, bounds.value.load) }
    }
    if (field === 'evenLoad' && typeof value === 'boolean') return { ...settings, evenLoad: value }
    if (typeof value !== 'number') return settings
    const within = bounds.value
    if (field === 'retention') return { ...settings, retention: held(value, within.retention) }
    if (field === 'newADay') return { ...settings, newADay: held(value, within.newADay) }
    if (field === 'reviewsADay') {
      return { ...settings, reviewsADay: held(value, within.reviewsADay) }
    }
    if (field === 'minutesADay') {
      return { ...settings, minutesADay: held(value, within.minutesADay) }
    }
    if (field === 'backlog') {
      return { ...settings, backlog: held(Math.round(value), within.backlog) }
    }
    if (field === 'interval') {
      return { ...settings, interval: held(Math.round(value), within.interval) }
    }
    return settings
  }

  /**
   * One field put where a person typed it, and marked theirs. A value the
   * field cannot hold moves nothing and claims nothing.
   */
  const moved = (one: OpenPreset, field: Field, value: SettingValue): void => {
    const was = one.settings.value
    one.settings.value = typed(was, field, value)
    if (one.settings.value !== was) one.theirs.add(field)
  }

  /** Where a value typed into the field the goal steers falls on the grid. */
  const falling = (one: OpenPreset, value: SettingValue): number => {
    if (typeof value === 'number') return nearest(one.curve.value.grid, value)
    if (typeof value === 'string' && isDay(value)) {
      return nearest(one.curve.value.grid, daysBetween(today(), value))
    }
    return -1
  }

  /** A goal of a date opens on a day, so one is named where the file names none. */
  const aiming = (settings: Settings, goal: Goal): Settings =>
    goal === 'date' && settings.byDate === ''
      ? { ...settings, goal, byDate: dayAfter(today(), AHEAD) }
      : { ...settings, goal }

  /** What one open preset holds, in the vocabulary its tab is drawn from. */
  const holding = (one: OpenPreset, id: string): PresetTabState => {
    const chooseGoal = (goal: Goal) => {
      const was = one.settings.value
      if (goal === was.goal) return
      one.settings.value = aiming(was, goal)
      one.theirs.add('goal')
      if (one.settings.value.byDate !== was.byDate) one.theirs.add('byDate')
      void writes(one)
      void curves(one)
    }

    const updateSetting = (field: Field, value: SettingValue) => {
      if (field === steers(one.settings.value.goal)) {
        moved(one, field, value)
        const place = falling(one, value)
        if (place >= 0) one.place.value = place
        if (shapeOf(one.settings.value) !== one.shape) void curves(one)
        return
      }
      moved(one, field, value)
      if (shapeOf(one.settings.value) !== one.shape) void curves(one)
    }

    const closeTab = (tab: string) => {
      void shut(one).then((gone) => {
        if (!gone) return
        open.delete(one.path.value)
        handle.closes(tab)
      })
    }

    return {
      id,
      settings: one.settings,
      curve: one.curve,
      counts: one.material,
      material: one.material,
      sliderPosition: one.place,
      place: one.place,
      isWaiting: one.waiting,
      waiting: one.waiting,
      bounds,
      problems: one.problems,
      stopped: one.stopped,
      errorMessage: one.saying,
      saying: one.saying,
      hasChanged: one.changed,
      changed: one.changed,
      again: () => void reads(one),
      reload: () => void reads(one),
      chooses: chooseGoal,
      chooseGoal,
      moves: (place) => turns(one, place),
      move: (place) => turns(one, place),
      moveSlider: (place) => turns(one, place),
      settles: () => void writes(one),
      save: () => void writes(one),
      types: updateSetting,
      updateSetting,
      shuts: closeTab,
      close: closeTab,
    }
  }

  /** What a preset tab holds, and nothing for a preset no tab has open. */
  const holds = (id: string): PresetTabState | undefined => {
    const one = open.get(id)
    return one && holding(one, id)
  }

  /** What a preset tab is called: the title the file carries, or the file itself. */
  const called = (path: string): string =>
    titles.get(path) || fileOf(path) || words.newPreset

  /**
   * A preset tab as the window keeps it. A preset is its own tab, under the
   * file it stands at, so the same preset asked for twice is the tab it has.
   */
  const kind: TabKind<PresetTabState, typeof PRESET> = {
    kind: PRESET,
    opens: (path) => {
      const one = keeps(path)
      open.set(path, one)
      void reads(one)
      return holding(one, path)
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
    void (showing === 'beside' ? handle.beside(PRESET, path) : handle.opens(PRESET, path))
  }

  // The editor of a preset, which is its one control and the settings under it.
  // A setting stands on no line of prose, so a preset opens whole.
  puts.holds('preset', shows)

  /** The vault changed: every open preset at one of those paths is read again. */
  const changed = (paths: readonly string[], renamed: readonly Move[] = []): void => {
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

  return { kind, holds, changed, called, shows, flush }
}
