/**
 * The keyboard, while the palette stands: what each key does, where the
 * keyboard goes as a step opens, and where it goes back to when the palette
 * closes.
 *
 * Everything the keys do not answer reaches the field, which is where a person
 * is typing.
 */
import { nextTick, onBeforeUnmount, onMounted, watch, type Ref, type ShallowRef } from 'vue'
import { stepTo } from '@numen/ui'
import { canLandOn, findKeptPlace, type PaletteAction } from '../lib/item'
import { isActionsChord } from '../lib/keys'
import type { PalettePlacesState } from './places'

/** The field, which takes the keyboard and holds it. */
export interface PaletteFieldHandle {
  readonly focus: () => void
  readonly select: () => void
}

export interface PaletteKeysOptions {
  readonly field: Readonly<ShallowRef<PaletteFieldHandle | null>>
  /** What is lit brought into sight. Only a key does this. */
  readonly reveal: () => void
  readonly places: PalettePlacesState
  readonly isPanelOpen: Ref<boolean>
  readonly getActions: () => readonly PaletteAction[]
  readonly typed: Ref<string>
  /** Whether the palette is drawn at all, and which step it is on. */
  readonly isOpen: () => boolean
  readonly getStep: () => string
  /** The item the keyboard relies on as a step opens. */
  readonly getOpensOn: () => string
  /** Where the keyboard goes back to once the palette closes. */
  readonly getOpenedFrom: () => HTMLElement | null
  /** Backspace in an empty field, and Escape. */
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
    context.places.chooseAt(context.places.here.value, event.shiftKey)
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
  const { field, places, isPanelOpen, typed } = options

  // The keyboard comes back to the field when the panel over it goes, and a
  // palette that is going takes it somewhere else itself.
  watch(isPanelOpen, async (now) => {
    if (now || !options.isOpen()) return
    await nextTick()
    field.value?.focus()
  })

  /** The keyboard moved through the places, and what it lands on brought into sight. */
  const stepBy = (delta: number, from = places.here.value): void => {
    places.goTo(stepTo(places.places.value, { from, by: delta }, canLandOn))
    options.reveal()
  }

  /** The action panel over the palette, when there is anything to put in it. */
  const openActions = (event: KeyboardEvent): void => {
    if (!options.getActions().length) return
    event.preventDefault()
    isPanelOpen.value = true
  }

  const onKey = (event: KeyboardEvent): void => {
    handlePaletteKey(event, { typed, places, isPanelOpen, options, stepBy, openActions })
  }

  /**
   * The keyboard put where the step wants it: on the value in force, else on
   * whatever it was standing on. Opening a step this way moves nothing, so a
   * caller that acts on what is lit acts on what is already so.
   */
  const enter = async (): Promise<void> => {
    isPanelOpen.value = false
    places.goTo(findKeptPlace(places.places.value, options.getOpensOn() || places.selectedId.value))
    await nextTick()
    field.value?.focus()
    field.value?.select()
    options.reveal()
  }

  const leave = (): void => {
    isPanelOpen.value = false
    places.selectedId.value = ''
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
