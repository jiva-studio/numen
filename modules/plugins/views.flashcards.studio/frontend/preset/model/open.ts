/**
 * Tab state and coordinator for an open flashcard preset.
 */
import { StopReason } from '@numen/protocol'
import { ref, shallowRef, type Ref } from 'vue'
import type { MessageWriter } from '@numen/plugin-sdk'
import {
  DEFAULTS,
  type Field,
  type Goal,
  type Presets,
  type PresetTabState,
  type Settings,
  type SettingsBounds,
  type SettingValue,
} from '../types'
import { produceSchedule } from '../lib/curve'
import { getShapeKey, getGoalField } from '../lib/fields'
import { WORDS as words } from '../words'
import { applyGoal, applyTypedSetting, findGridIndex, reconcileSettings } from '../lib/settings'
import { createCurveState, updateCurves, type CurveState } from './curves'
import { canCloseTab, createWriteQueue, requestWrite, type WriteQueue } from './writeQueue'

/** State held by one open preset tab. */
export interface OpenPreset {
  readonly path: Ref<string>
  readonly settings: Ref<Settings>
  readonly problems: Ref<readonly string[]>
  readonly stopped: Ref<StopReason>
  readonly curves: CurveState
  readonly writeQueue: WriteQueue
}

export function createOpenPreset(path: string, today: string, bounds: SettingsBounds): OpenPreset {
  return {
    path: ref(path),
    settings: shallowRef<Settings>(DEFAULTS),
    problems: shallowRef<readonly string[]>([]),
    stopped: ref<StopReason>(StopReason.NOTHING),
    curves: createCurveState(DEFAULTS, today, bounds),
    writeQueue: createWriteQueue(),
  }
}

export interface ReadPresetContext {
  core: Presets
  bounds: Ref<SettingsBounds>
  titles: Map<string, string>
  today: string
}

/** Reads the preset settings and curve from the vault. */
export const readPreset = async (one: OpenPreset, context: ReadPresetContext): Promise<void> => {
  const { core, bounds, titles, today } = context
  let answer
  try {
    answer = await core.read(one.path.value)
  } catch {
    // The window says what it could not do; what the call carried back adds nothing a person can act on.
    one.writeQueue.errorMessage.value = words.unreachable
    one.curves.isWaiting.value = false
    return
  }
  one.writeQueue.errorMessage.value = answer.ok ? '' : words.notRead(answer.error)
  one.writeQueue.hasExternalChange.value = false
  one.curves.answers.clear()
  const readItem = answer.ok ? answer.value.preset : null
  if (answer.ok) {
    one.writeQueue.revision = answer.value.at
    bounds.value = answer.value.bounds
  }
  if (!readItem) {
    one.problems.value = []
    one.stopped.value = StopReason.NOTHING
    one.curves.isWaiting.value = false
    return
  }
  if (readItem.title) titles.set(one.path.value, readItem.title)
  one.problems.value = readItem.problems
  one.stopped.value = readItem.stopsOn
  one.settings.value = reconcileSettings(
    readItem.settings,
    one.settings.value,
    one.writeQueue.dirtyFields,
  )
  await updateCurves(one.curves, {
    path: one.path.value,
    getSettings: () => one.settings.value,
    core,
    bounds: bounds.value,
    today,
    setErrorMessage: (msg) => {
      one.writeQueue.errorMessage.value = msg
    },
  })
}

export interface PresetStateContext {
  closeTab: (id: string) => void
  onClosed: (path: string) => void
  core: Presets
  bounds: Ref<SettingsBounds>
  writeMessage: MessageWriter
  today: () => string
  titles: Map<string, string>
}

const createGoalChooser = (one: OpenPreset, context: PresetStateContext) => (goal: Goal) => {
  const was = one.settings.value
  if (goal === was.goal) return
  one.settings.value = applyGoal(was, goal, context.today())
  one.writeQueue.dirtyFields.add('goal')
  if (one.settings.value.byDate !== was.byDate) one.writeQueue.dirtyFields.add('byDate')
  void requestWrite(one.writeQueue, {
    path: one.path.value,
    settings: one.settings.value,
    core: context.core,
    writeMessage: context.writeMessage,
  })
  void updateCurves(one.curves, {
    path: one.path.value,
    getSettings: () => one.settings.value,
    core: context.core,
    bounds: context.bounds.value,
    today: context.today(),
    setErrorMessage: (msg) => {
      one.writeQueue.errorMessage.value = msg
    },
  })
}

const createSettingUpdater =
  (one: OpenPreset, context: PresetStateContext) => (field: Field, value: SettingValue) => {
    const was = one.settings.value
    one.settings.value = applyTypedSetting(was, { field, value }, context.bounds.value)
    if (one.settings.value !== was) one.writeQueue.dirtyFields.add(field)

    if (field === getGoalField(one.settings.value.goal)) {
      const place = findGridIndex(one.curves.curve.value.grid, value, context.today())
      if (place >= 0) one.curves.place.value = place
    }
    if (getShapeKey(one.settings.value) !== one.curves.shape) {
      void updateCurves(one.curves, {
        path: one.path.value,
        getSettings: () => one.settings.value,
        core: context.core,
        bounds: context.bounds.value,
        today: context.today(),
        setErrorMessage: (msg) => {
          one.writeQueue.errorMessage.value = msg
        },
      })
    }
  }

const createSliderMover = (one: OpenPreset, context: PresetStateContext) => (place: number) => {
  const was = one.settings.value
  one.settings.value = produceSchedule(was, place, {
    curve: one.curves.curve.value,
    today: context.today(),
    within: context.bounds.value,
  })
  one.curves.place.value = place
  one.writeQueue.dirtyFields.add(getGoalField(one.curves.curve.value.goal))
}

const createTabCloser = (one: OpenPreset, context: PresetStateContext) => (tab: string) => {
  void canCloseTab(one.writeQueue, {
    path: one.path.value,
    settings: one.settings.value,
    core: context.core,
    writeMessage: context.writeMessage,
  }).then((removed) => {
    if (!removed) return
    context.onClosed(one.path.value)
    context.closeTab(tab)
  })
}

/** Creates the public tab state interface for an open preset. */
export const createPresetState = (
  one: OpenPreset,
  id: string,
  context: PresetStateContext,
): PresetTabState => ({
  id,
  settings: one.settings,
  curve: one.curves.curve,
  material: one.curves.material,
  place: one.curves.place,
  isWaiting: one.curves.isWaiting,
  bounds: context.bounds,
  problems: one.problems,
  stopped: one.stopped,
  errorMessage: one.writeQueue.errorMessage,
  hasExternalChange: one.writeQueue.hasExternalChange,
  reload: () =>
    void readPreset(one, {
      core: context.core,
      bounds: context.bounds,
      titles: context.titles,
      today: context.today(),
    }),
  chooseGoal: createGoalChooser(one, context),
  moveSlider: createSliderMover(one, context),
  flushPending: () =>
    void requestWrite(one.writeQueue, {
      path: one.path.value,
      settings: one.settings.value,
      core: context.core,
      writeMessage: context.writeMessage,
    }),
  updateSetting: createSettingUpdater(one, context),
  close: createTabCloser(one, context),
})
