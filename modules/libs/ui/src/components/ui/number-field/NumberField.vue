<script setup lang="ts">
/**
 * One line of digits, typed by hand and held inside its bounds, worked as a
 * spin button.
 *
 * What was typed stands as it was typed, and the number is written out only
 * once the field is left. Bounds that move under the number bring it in.
 */
import { computed, nextTick, ref, useTemplateRef, watch, type HTMLAttributes } from 'vue'
import { cn } from '@/classes'
import {
  allowed,
  clamped,
  numberOf,
  onItsWay,
  settled,
  standsFor,
  walked,
  written,
  DEFAULT_BOUNDS,
} from './number'

const props = withDefaults(
  defineProps<{
    /** How far the number may go. */
    min?: number
    max?: number
    /** What an arrow key moves it by. */
    step?: number
    disabled?: boolean
    /** The words standing in for a number nobody has typed. */
    placeholder?: string
    class?: HTMLAttributes['class']
  }>(),
  {
    min: DEFAULT_BOUNDS.min,
    max: DEFAULT_BOUNDS.max,
    step: DEFAULT_BOUNDS.step,
    disabled: false,
    placeholder: '',
  },
)

/** The number in force. An empty field holds none. */
const model = defineModel<number | null>({ default: null })

const raises = defineEmits<{
  /** The field come to rest at a number other than the one it was resting at. */
  settles: [value: number | null]
}>()

const bounds = computed(() => ({ min: props.min, max: props.max, step: props.step }))

/** What stands in the field, which is what was typed until the field is left. */
const typed = ref(written(model.value))

/** The number the field last stood at rest at. Settling is what moves it. */
let rested = model.value

/**
 * Whether what stands there is refused. Text a number could still be typed out
 * of is left alone; text that is no number, and a number past the bounds, are
 * marked where they stand. A number between two places the step lays is a
 * number on the way to one of them, and stands unmarked until the field is left.
 */
const refused = computed(() => {
  const said = typed.value.trim()
  if (said === '') return false
  if (!onItsWay(said)) return true
  const value = numberOf(said)
  return value !== null && value !== clamped(value, bounds.value)
})

/** What a refused line is said as, so it is read out and not only marked. */
const saying = computed(() => (refused.value ? typed.value.trim() : undefined))

/** The number in force, which is a number the bounds hold. */
const inForce = computed(() =>
  model.value === null ? null : clamped(model.value, bounds.value),
)

/** A number the bounds no longer hold is brought in, and stands there written out. */
watch(
  inForce,
  (now) => {
    if (now === model.value) return
    model.value = now
    typed.value = written(now)
    rested = now
  },
  { immediate: true },
)

const element = useTemplateRef<HTMLInputElement>('element')

/** A number set from outside is written out; typing that means it is left alone. */
watch(model, (now) => {
  if (standsFor(typed.value, now)) return
  typed.value = written(now)
  rested = now
})

const took = (event: Event) => {
  typed.value = (event.target as HTMLInputElement).value
  if (typed.value.trim() === '') model.value = null
  else if (allowed(typed.value, bounds.value)) model.value = numberOf(typed.value)
}

/**
 * Leaving writes back what holds: the number on a place of the step, in bounds.
 * A field left standing where it stood has not settled anywhere new.
 *
 * What the caller does with the number is the caller's, and the field stands on
 * whatever the caller holds once it has answered.
 */
const settle = async () => {
  const value = numberOf(typed.value)
  const now = value === null ? null : settled(value, bounds.value)
  model.value = now
  typed.value = written(now)
  if (now !== rested) {
    rested = now
    raises('settles', now)
  }
  await nextTick()
  if (!standsFor(typed.value, model.value)) {
    typed.value = written(model.value)
    rested = model.value
  }
}

/**
 * A key the spin button answers: the arrows a step, the page keys ten, and home
 * and end the ends. Enter settles the field where it stands.
 */
const pressed = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    void settle()
    return
  }
  const said = walked(event.key, numberOf(typed.value) ?? model.value, bounds.value)
  if (said === null) return
  event.preventDefault()
  if (props.disabled) return
  model.value = said
  typed.value = written(said)
}

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  element,
  focus: (how?: FocusOptions) => element.value?.focus(how),
})
</script>

<template>
  <input
    ref="element"
    type="text"
    inputmode="decimal"
    autocomplete="off"
    role="spinbutton"
    data-slot="number-field"
    :value="typed"
    :disabled="disabled"
    :placeholder="placeholder"
    :aria-valuemin="min"
    :aria-valuemax="max"
    :aria-valuenow="model ?? undefined"
    :aria-valuetext="saying"
    :aria-invalid="refused || undefined"
    :class="
      cn(
        'w-full rounded-tight border border-field-rule bg-field',
        // One row tall, which every control standing on a row is drawn at.
        'h-action px-2',
        'font-sans text-base leading-none text-ink tabular-nums placeholder:text-hushed',
        'outline-none ring-numen',
        'aria-invalid:border-alarm aria-invalid:text-alarm',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @input="took"
    @blur="settle"
    @keydown="pressed"
  />
</template>
