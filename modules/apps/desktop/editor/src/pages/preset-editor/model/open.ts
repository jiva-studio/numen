/**
 * Tab state and coordinator for an open flashcard preset.
 */
import { StopReason } from '@numen/protocol'
import { ref, shallowRef, type Ref } from 'vue'
import type { WindowHandle } from '@/entities/tab'
import type { MessageWriter } from '@/shared/notices/messages'
import { DEFAULTS } from '../types'
import { produceSchedule } from '../lib/curve'
import { shapeOf, steer } from '../lib/fields'
import { WORDS as words } from '../words'
import type {
  Field,
  Goal,
  Presets,
  PresetTabState,
  Settings,
  SettingsBounds,
  SettingValue,
} from '../types'
import { aimGoal, applyTypedSetting, findGridIndex, reconcileSettings } from '../lib/settings'
import { createCurveState, updateCurves, type CurveState } from './curves'
import { canCloseTab, createWriteFlight, requestWrite, type WriteFlight } from './flight'

/** State held by one open preset tab. */
export interface OpenPreset {
  readonly path: Ref<string>
  readonly settings: Ref<Settings>
  readonly problems: Ref<readonly string[]>
  readonly stopped: Ref<StopReason>
  readonly curves: CurveState
  readonly flight: WriteFlight
}

export function createOpenPreset(path: string, today: string, bounds: SettingsBounds): OpenPreset {
  return {
    path: ref(path),
    settings: shallowRef<Settings>(DEFAULTS),
    problems: shallowRef<readonly string[]>([]),
    stopped: ref<StopReason>(StopReason.NOTHING),
    curves: createCurveState(DEFAULTS, today, bounds),
    flight: createWriteFlight(),
  }
}

/** Reads the preset settings and curve from the vault. */
export const readPreset = async (
  one: OpenPreset,
  core: Presets,
  bounds: Ref<SettingsBounds>,
  titles: Map<string, string>,
  today: string,
): Promise<void> => {
  let answer
  try {
    answer = await core.read(one.path.value)
  } catch {
    // The window says what it could not do; what the call carried back adds nothing a person can act on.
    one.flight.errorMessage.value = words.unreachable
    one.curves.isWaiting.value = false
    return
  }
  one.flight.errorMessage.value = answer.ok ? '' : words.notRead(answer.error)
  one.flight.hasChanged.value = false
  one.curves.answers.clear()
  const read = answer.ok ? answer.value.preset : null
  if (answer.ok) {
    one.flight.at = answer.value.at
    bounds.value = answer.value.bounds
  }
  if (!read) {
    one.problems.value = []
    one.stopped.value = StopReason.NOTHING
    one.curves.isWaiting.value = false
    return
  }
  if (read.title) titles.set(one.path.value, read.title)
  one.problems.value = read.problems
  one.stopped.value = read.stopsOn
  one.settings.value = reconcileSettings(read.settings, one.settings.value, one.flight.theirs)
  await updateCurves(
    one.curves,
    one.path.value,
    () => one.settings.value,
    core,
    bounds.value,
    today,
    (msg) => {
      one.flight.errorMessage.value = msg
    },
  )
}

/** Creates the public tab state interface for an open preset. */
export const createPresetState = (
  one: OpenPreset,
  id: string,
  handle: WindowHandle,
  onClosed: (path: string) => void,
  core: Presets,
  bounds: Ref<SettingsBounds>,
  writeMessage: MessageWriter,
  today: () => string,
  titles: Map<string, string>,
): PresetTabState => {
  const chooseGoal = (goal: Goal) => {
    const was = one.settings.value
    if (goal === was.goal) return
    one.settings.value = aimGoal(was, goal, today())
    one.flight.theirs.add('goal')
    if (one.settings.value.byDate !== was.byDate) one.flight.theirs.add('byDate')
    void requestWrite(one.flight, one.path.value, one.settings.value, core, writeMessage)
    void updateCurves(
      one.curves,
      one.path.value,
      () => one.settings.value,
      core,
      bounds.value,
      today(),
      (msg) => {
        one.flight.errorMessage.value = msg
      },
    )
  }

  const updateSetting = (field: Field, value: SettingValue) => {
    const was = one.settings.value
    one.settings.value = applyTypedSetting(was, field, value, bounds.value)
    if (one.settings.value !== was) one.flight.theirs.add(field)

    if (field === steer(one.settings.value.goal)) {
      const place = findGridIndex(one.curves.curve.value.grid, value, today())
      if (place >= 0) one.curves.place.value = place
    }
    if (shapeOf(one.settings.value) !== one.curves.shape) {
      void updateCurves(
        one.curves,
        one.path.value,
        () => one.settings.value,
        core,
        bounds.value,
        today(),
        (msg) => {
          one.flight.errorMessage.value = msg
        },
      )
    }
  }

  const moveSlider = (place: number) => {
    const was = one.settings.value
    one.settings.value = produceSchedule(was, place, one.curves.curve.value, today(), bounds.value)
    one.curves.place.value = place
    one.flight.theirs.add(steer(one.curves.curve.value.goal))
  }

  const closeTab = (tab: string) => {
    void canCloseTab(one.flight, one.path.value, one.settings.value, core, writeMessage).then(
      (gone) => {
        if (!gone) return
        onClosed(one.path.value)
        handle.closeTab(tab)
      },
    )
  }

  return {
    id,
    settings: one.settings,
    curve: one.curves.curve,
    material: one.curves.material,
    place: one.curves.place,
    isWaiting: one.curves.isWaiting,
    bounds,
    problems: one.problems,
    stopped: one.stopped,
    errorMessage: one.flight.errorMessage,
    hasChanged: one.flight.hasChanged,
    reload: () => void readPreset(one, core, bounds, titles, today()),
    chooseGoal,
    moveSlider,
    settle: () =>
      void requestWrite(one.flight, one.path.value, one.settings.value, core, writeMessage),
    updateSetting,
    close: closeTab,
  }
}
