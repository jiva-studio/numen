/**
 * Keyboard handling while the palette is open: the action of each key, where
 * focus moves when a step opens, and where focus returns when the palette
 * closes.
 *
 * Keys without a palette action go to the field, where the user types.
 */
import { nextTick, onBeforeUnmount, onMounted, watch, type Ref, type ShallowRef } from 'vue'
import { stepTo } from '@numen/ui'
import { canLandOn, findKeptPlace, type PaletteAction } from '../lib/item'
import { isActionsChord } from '../lib/keys'
import type { ActiveItemState } from './activeItem'
import type { PalettePlacesState } from './places'

/** The field, which takes the keyboard and holds it. */
export interface PaletteFieldHandle {
  readonly focus: () => void
  readonly select: () => void
}

export interface PaletteKeysOptions {
  readonly field: Readonly<ShallowRef<PaletteFieldHandle | null>>
  readonly reveal: () => void
  readonly places: PalettePlacesState
  readonly active: ActiveItemState
  readonly isPanelOpen: Ref<boolean>
  readonly getActions: () => readonly PaletteAction[]
  readonly typed: Ref<string>
  readonly isOpen: () => boolean
  readonly getStep: () => string
  readonly getOpensOn: () => string
  readonly getOpenedFrom: () => HTMLElement | null
  readonly goBack: () => void
  readonly dismiss: () => void
}

export interface PaletteKeysState {
  readonly onKey: (event: KeyboardEvent) => void
}

/** How far each step key moves, and the place it counts from. */
const STEPS: Readonly<Record<string, readonly [by: number, from?: number]>> = {
  ArrowDown: [1],
  ArrowUp: [-1],
  Home: [1, -1],
  End: [-1, 0],
}

interface KeyContext {
  readonly typed: Readonly<{ value: string }>
  readonly places: PalettePlacesState
  readonly active: ActiveItemState
  readonly isPanelOpen: { value: boolean }
  readonly options: PaletteKeysOptions
  readonly stepBy: (delta: number, from?: number) => void
  readonly openActions: (event: KeyboardEvent) => void
}

function handlePaletteKey(event: KeyboardEvent, context: KeyContext): void {
  if (isActionsChord(event)) {
    context.openActions(event)
    return
  }

  const step = STEPS[event.key]
  if (step) {
    event.preventDefault()
    context.stepBy(step[0], step[1])
    return
  }

  if (event.key === 'Enter') {
    event.preventDefault()
    const pos =
      context.active.here.value >= 0
        ? context.active.here.value
        : stepTo(context.places.places.value, { from: -1, by: 1 }, canLandOn)
    context.active.chooseAt(pos, event.shiftKey)
    return
  }

  if (event.key === 'Backspace' && context.typed.value === '') {
    event.preventDefault()
    context.options.goBack()
    return
  }

  if (event.key === 'Escape') {
    event.preventDefault()
    context.options.dismiss()
    return
  }

  if (event.key === 'Tab') event.preventDefault()
}

export function usePaletteKeys(options: PaletteKeysOptions): PaletteKeysState {
  const { field, places, active, isPanelOpen, typed } = options

  // The keyboard returns to the field when the panel over it goes, and a
  // palette that is going takes it somewhere else itself.
  watch(isPanelOpen, async (now) => {
    if (now || !options.isOpen()) return
    await nextTick()
    field.value?.focus()
  })

  /** The keyboard moved through the places, and what it lands on brought into sight. */
  const stepBy = (delta: number, from = active.here.value): void => {
    active.goTo(stepTo(places.places.value, { from, by: delta }, canLandOn))
    options.reveal()
  }

  /** The action panel over the palette, when there is anything to put in it. */
  const openActions = (event: KeyboardEvent): void => {
    if (!options.getActions().length) return
    event.preventDefault()
    isPanelOpen.value = true
  }

  const onKey = (event: KeyboardEvent): void => {
    handlePaletteKey(event, { typed, places, active, isPanelOpen, options, stepBy, openActions })
  }

  /**
   * Moves focus to where the step requires: the current value if there is one,
   * else the last focused item. Opening a step this way changes no selection, so a
   * caller that acts on the highlighted item acts on the item already
   * highlighted.
   */
  const enter = async (): Promise<void> => {
    isPanelOpen.value = false
    active.goTo(findKeptPlace(places.places.value, options.getOpensOn() || active.selectedId.value))
    await nextTick()
    field.value?.focus()
    field.value?.select()
    options.reveal()
  }

  const leave = (): void => {
    isPanelOpen.value = false
    active.selectedId.value = ''
    const back = options.getOpenedFrom()
    if (back?.isConnected) back.focus()
  }

  watch(options.isOpen, async (now) => {
    if (now) await enter()
    else leave()
  })

  watch(options.getStep, async () => {
    if (options.isOpen()) await enter()
  })

  onMounted(async () => {
    if (options.isOpen()) await enter()
  })

  // A palette can go while it is still open, and the keyboard goes back with it.
  onBeforeUnmount(() => {
    if (options.isOpen()) leave()
  })

  return { onKey }
}
