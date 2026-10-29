<script setup lang="ts">
/**
 * A single-line digit input, typed manually and clamped to its bounds, behaving as a
 * spin button.
 *
 * The text stays as typed, and the number is written out only when the field loses
 * focus. If the bounds change under the number, it is clamped to them.
 */
import { computed, useId, useTemplateRef } from 'vue'
import { cn } from '@/shared/lib/dom/classes'
import { DEFAULT_BOUNDS, useNumberEntry } from './number'
import type { NumberFieldProps } from './NumberField.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<NumberFieldProps>(), {
  min: DEFAULT_BOUNDS.min,
  max: DEFAULT_BOUNDS.max,
  step: DEFAULT_BOUNDS.step,
  isDisabled: false,
  id: '',
  placeholder: '',
})

/** The number in force. An empty field holds none. */
const model = defineModel<number | null>({ default: null })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The field come to rest at a number other than the one it was resting at. */
  commit: [value: number | null]
}>()

/* ---------------------------------- State --------------------------------- */
const bounds = computed(() => ({ min: props.min, max: props.max, step: props.step }))

const { typed, isInvalid, speechText, setText, commit, applyKey } = useNumberEntry(
  model,
  () => bounds.value,
  (value) => emit('commit', value),
)

const ownId = useId()

const element = useTemplateRef<HTMLInputElement>('element')

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  element,
  focus: (how?: FocusOptions) => element.value?.focus(how),
})

/* -------------------------------- Handlers -------------------------------- */
const onInput = (event: Event) => setText((event.target as HTMLInputElement).value)

/**
 * A key the spin button answers: the arrows a step, the page keys ten, and home
 * and end the ends. Enter commits the field at its current position.
 */
const onKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    void commit()
    return
  }
  if (applyKey(event.key, props.isDisabled)) event.preventDefault()
}
</script>

<template>
  <input
    ref="element"
    :id="id || ownId"
    type="text"
    inputmode="decimal"
    autocomplete="off"
    role="spinbutton"
    data-slot="number-field"
    :value="typed"
    :disabled="isDisabled"
    :placeholder="placeholder"
    :aria-valuemin="min"
    :aria-valuemax="max"
    :aria-valuenow="model ?? undefined"
    :aria-valuetext="speechText"
    :aria-invalid="isInvalid || undefined"
    :class="
      cn(
        'rounded-tight border-field-rule bg-field w-full border',
        // One row tall, which every control standing on a row is drawn at.
        'h-action px-2',
        'text-ink placeholder:text-hushed font-sans text-base leading-none tabular-nums',
        'ring-numen outline-none',
        'aria-invalid:border-alarm aria-invalid:text-alarm',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @input="onInput"
    @blur="commit"
    @keydown="onKeyDown"
  />
</template>
