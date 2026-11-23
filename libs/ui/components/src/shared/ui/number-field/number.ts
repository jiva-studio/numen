import { computed, nextTick, ref, watch, type ComputedRef, type Ref } from 'vue'
import { clamp, type Bounds } from '@/shared/lib/math/bounds'
import {
  formatNumber,
  isAllowed,
  isPartialNumber,
  isTextForValue,
  parseNumber,
  stepForKey,
} from './numberText'
import { fitToBounds } from './fitToBounds'

export const DEFAULT_BOUNDS: Bounds = { min: 0, max: 999, step: 1 }

/** The field's state while the user types, and what each action does with it. */
export interface NumberEntry {
  readonly typed: Ref<string>
  readonly isInvalid: ComputedRef<boolean>
  readonly speechText: ComputedRef<string | undefined>
  readonly setText: (text: string) => void
  readonly commit: () => Promise<void>
  readonly applyKey: (key: string, isDisabled: boolean) => boolean
}

const isInvalidNumber = (raw: string, getBounds: () => Bounds): boolean => {
  if (!isPartialNumber(raw)) return true
  const value = parseNumber(raw)
  // Stryker disable next-line ConditionalExpression: value !== null narrows number | null for clamp
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

  /** The number the field last settled on. Committing updates it. */
  let rested = model.value

  const isInvalid = computed(() => isInvalidNumber(formattedValue.value, getBounds))
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
   * Leaving the field writes back the number snapped to the step and the bounds.
   * A field left at its previous number commits nothing new. Afterwards the field
   * shows the value the caller holds once it has handled the commit.
   */
  const commit = async (): Promise<void> => {
    const value = parseNumber(formattedValue.value)
    const now = value === null ? null : fitToBounds(value, getBounds())
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
