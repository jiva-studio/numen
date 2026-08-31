<script setup lang="ts">
/**
 * One line of digits, typed by hand and held inside its bounds.
 *
 * What was typed stands as it was typed: a line that is not a number in the
 * bounds is marked and hands nothing on, and the number in force is written
 * back only once the field is left.
 *
 * Typing and leaving are two things said, so a caller can follow the digits
 * and act on the number the field comes to rest at.
 */
import { computed, ref, useTemplateRef, watch, type HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import {
  allowed,
  clamped,
  numberOf,
  onItsWay,
  stepped,
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
  /** The field left, at the number that stands there once it is settled. */
  settles: [value: number | null]
}>()

const bounds = computed(() => ({ min: props.min, max: props.max, step: props.step }))

/** What stands in the field, which is what was typed until the field is left. */
const typed = ref(written(model.value))

/**
 * Whether what stands there is refused. Text a number could still be typed out
 * of is left alone; text that is no number, and a number past the bounds, are
 * marked where they stand.
 */
const refused = computed(() => {
  const said = typed.value.trim()
  if (said === '') return false
  if (!onItsWay(said)) return true
  const value = numberOf(said)
  return value !== null && value !== clamped(value, bounds.value)
})

const element = useTemplateRef<HTMLInputElement>('element')

/** A number set from outside is written out; typing that means it is left alone. */
watch(model, (now) => {
  if (numberOf(typed.value) !== now) typed.value = written(now)
})

const took = (event: Event) => {
  typed.value = (event.target as HTMLInputElement).value
  if (typed.value.trim() === '') model.value = null
  else if (allowed(typed.value, bounds.value)) model.value = numberOf(typed.value)
}

/** Leaving writes back what holds: the number brought inside the bounds. */
const settle = () => {
  const value = numberOf(typed.value)
  model.value = value === null ? null : clamped(value, bounds.value)
  typed.value = written(model.value)
  raises('settles', model.value)
}

const move = (by: number) => {
  if (props.disabled) return
  model.value = stepped(numberOf(typed.value) ?? model.value, by, bounds.value)
  typed.value = written(model.value)
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
    :aria-invalid="refused || undefined"
    :class="
      cn(
        'w-full rounded-tight border border-field-rule bg-field',
        // The gaps are measured to the ink the screen paints. A field keeps a
        // line box of its own, taller than the digits in it, so the block
        // padding is set under the inline one and the box is one row tall.
        'px-2 py-1.5',
        'font-sans text-base leading-none text-ink tabular-nums placeholder:text-hushed',
        'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
        'aria-invalid:border-alarm aria-invalid:text-alarm',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @input="took"
    @blur="settle"
    @keydown.up.prevent="move(1)"
    @keydown.down.prevent="move(-1)"
    @keydown.enter="settle"
  />
</template>
