/**
 * What a line of typing evaluates to as a number, where the bounds leave it, and
 * where a key leaves it.
 *
 * Pure: the same text gives the same answer wherever it is read, so what the
 * field hands on can be named without drawing it.
 */
import { computed, nextTick, ref, watch, type ComputedRef, type Ref } from 'vue'
import { clamp, getDecimalPlaces, roundToPlaces, type Bounds } from '@/shared/lib/bounds'

export const DEFAULT_BOUNDS: Bounds = { min: 0, max: 999, step: 1 }

/** Text a number could still be typed out of: a lone sign, a trailing point. */
const ON_ITS_WAY = /^[+-]?(\d+([.,]\d*)?|[.,]\d*)?$/

/** A comma between a whole number and a group of three, which is no point. */
const GROUPED = /^[+-]?[1-9]\d{0,2},\d{3}$/

/** Whether more typing could still make a number of the text. */
export const isPartialNumber = (text: string): boolean => {
  const cleanText = text.trim()
  return ON_ITS_WAY.test(cleanText) && !GROUPED.test(cleanText)
}

/**
 * The number the text stands for; nothing at its current position for none. The shape
 * is what a person types: no exponent, no hexadecimal, no word for infinity.
 */
export const parseNumber = (text: string): number | null => {
  const cleanText = text.trim()
  if (cleanText === '' || !isPartialNumber(cleanText)) return null
  const value = Number(cleanText.replace(',', '.'))
  return Number.isFinite(value) ? value : null
}

/** The place the step lays nearest a number, counted from the floor. */
const onStep = (value: number, bounds: Bounds): number => {
  if (bounds.step <= 0) return value
  const steps = Math.round((value - bounds.min) / bounds.step)
  return roundToPlaces(bounds.min + steps * bounds.step, getDecimalPlaces(bounds.step))
}

/** Where the bounds leave a number: on a place the step lays, inside the ends. */
export const snapToBounds = (value: number, bounds: Bounds): number =>
  clamp(onStep(value, bounds), bounds)

/** Whether the text is a number the bounds allow. An empty field is neither. */
export const isAllowed = (text: string, bounds: Bounds): boolean => {
  const value = parseNumber(text)
  return value !== null && value === snapToBounds(value, bounds)
}

/** Where an arrow key leaves the number: one step from at its current position. */
export const stepBy = (value: number | null, by: number, bounds: Bounds): number =>
  snapToBounds((value ?? bounds.min) + by * bounds.step, bounds)

/** How many steps a page key covers at once. */
const PACES = 10

/** Where a key leaves the number, and nothing for a key the field does not answer. */
export const stepForKey = (key: string, value: number | null, bounds: Bounds): number | null => {
  if (key === 'Home') return snapToBounds(bounds.min, bounds)
  if (key === 'End') return snapToBounds(bounds.max, bounds)
  if (key === 'ArrowUp') return stepBy(value, 1, bounds)
  if (key === 'ArrowDown') return stepBy(value, -1, bounds)
  if (key === 'PageUp') return stepBy(value, PACES, bounds)
  if (key === 'PageDown') return stepBy(value, -PACES, bounds)
  return null
}

/** How a number is written into the field. */
export const formatNumber = (value: number | null): string => (value === null ? '' : String(value))

/** Whether what is typed stands for the number in force. An empty field holds none. */
export const isTextForValue = (text: string, value: number | null): boolean =>
  value === null ? text.trim() === '' : parseNumber(text) === value

/** What the field relies on while a person types, and what each act does with it. */
export interface NumberEntry {
  readonly typed: Ref<string>
  /** Whether what stands there is refused. */
  readonly isInvalid: ComputedRef<boolean>
  /** What a refused line is said as, so it is read out and not only marked. */
  readonly speechText: ComputedRef<string | undefined>
  /** The text now in the field. */
  readonly setText: (text: string) => void
  /** The field left, or committed with Enter. */
  readonly commit: () => Promise<void>
  /** A key pressed; whether it was one the spin button answers. */
  readonly applyKey: (key: string, isDisabled: boolean) => boolean
}

/**
 * The typed, model and rested state of a number field.
 *
 * What was typed stands as it was typed, and the number is written out only
 * once the field is left. A number the bounds no longer hold is brought in, and
 * stands there written out. `onCommit` hears of a rest at a number other than
 * the one the field was resting at.
 */
const checkInvalid = (raw: string, getBounds: () => Bounds): boolean => {
  const text = raw.trim()
  if (text === '') return false
  if (!isPartialNumber(text)) return true
  const value = parseNumber(text)
  return value !== null && value !== clamp(value, getBounds())
}

interface NumberSyncState {
  readonly model: Ref<number | null>
  readonly inForce: ComputedRef<number | null>
  readonly typed: Ref<string>
}

const watchNumberSync = (
  state: NumberSyncState,
  setRested: (value: number | null) => void,
): void => {
  const { model, inForce, typed } = state
  watch(
    inForce,
    (now) => {
      if (now === model.value) return
      model.value = now
      typed.value = formatNumber(now)
      setRested(now)
    },
    { immediate: true },
  )

  watch(model, (now) => {
    if (isTextForValue(typed.value, now)) return
    typed.value = formatNumber(now)
    setRested(now)
  })
}

export const useNumberEntry = (
  model: Ref<number | null>,
  getBounds: () => Bounds,
  onCommit: (value: number | null) => void,
): NumberEntry => {
  const formattedValue = ref(formatNumber(model.value))

  /** The number the field last stood at rest at. Committing is what moves it. */
  let rested = model.value

  const isInvalid = computed(() => checkInvalid(formattedValue.value, getBounds))
  const speechText = computed(() => (isInvalid.value ? formattedValue.value.trim() : undefined))
  const inForce = computed(() => (model.value === null ? null : clamp(model.value, getBounds())))

  watchNumberSync({ model, inForce, typed: formattedValue }, (now) => {
    rested = now
  })

  const setText = (text: string): void => {
    formattedValue.value = text
    if (text.trim() === '') model.value = null
    else if (isAllowed(text, getBounds())) model.value = parseNumber(text)
  }

  /**
   * Leaving writes back what holds: the number on a place of the step, in
   * bounds. A field left standing where it stood has not committed anywhere new.
   * The field relies on whatever the caller holds once it has answered.
   */
  const commit = async (): Promise<void> => {
    const value = parseNumber(formattedValue.value)
    const now = value === null ? null : snapToBounds(value, getBounds())
    model.value = now
    formattedValue.value = formatNumber(now)
    if (now !== rested) {
      rested = now
      onCommit(now)
    }
    await nextTick()
    if (!isTextForValue(formattedValue.value, model.value)) {
      formattedValue.value = formatNumber(model.value)
      rested = model.value
    }
  }

  const applyKey = (key: string, isDisabled: boolean): boolean => {
    const stepped = stepForKey(key, parseNumber(formattedValue.value) ?? model.value, getBounds())
    if (stepped === null) return false
    if (isDisabled) return true
    model.value = stepped
    formattedValue.value = formatNumber(stepped)
    return true
  }

  return { typed: formattedValue, isInvalid, speechText, setText, commit, applyKey }
}
