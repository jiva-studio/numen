<script setup lang="ts">
/**
 * An hour and a minute of the day, written as `04:00` and typed on the clock
 * the machine draws.
 *
 * A field standing at an hour of the day hands that hour on. A field standing
 * at half an hour, or at nothing, hands nothing on and is left at its current position.
 */
import { computed, useId, useTemplateRef, type HTMLAttributes } from 'vue'
import { cn } from '@/shared/lib/classes'
import { isClockTime } from './clock'

const props = withDefaults(
  defineProps<{
    /** What a `<label for>` names the field by. Left empty, the field makes up its own. */
    id?: string
    /** How early and how late in the day the hour may stand, written as `04:00`. */
    min?: string
    max?: string
    isDisabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { id: '', min: '', max: '', isDisabled: false },
)

/** The hour in force, written as `04:00`. An empty field holds none. */
const model = defineModel<string>({ default: '' })

const emit = defineEmits<{
  /** The field come to rest at an hour of the day. */
  commit: [value: string]
}>()

const inForce = computed(() => (isClockTime(model.value) ? model.value : ''))

const ownId = useId()

const element = useTemplateRef<HTMLInputElement>('element')

const onChange = (event: Event) => {
  const nextValue = (event.target as HTMLInputElement).value
  if (!isClockTime(nextValue) || nextValue === model.value) return
  model.value = nextValue
  emit('commit', nextValue)
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
    :id="id || ownId"
    type="time"
    data-slot="time-field"
    :value="inForce"
    :disabled="isDisabled"
    :min="min || undefined"
    :max="max || undefined"
    :class="
      cn(
        'time-field rounded-tight border-field-rule bg-field w-full border',
        // One row tall, which every control standing on a row is drawn at.
        'h-action px-2',
        'text-ink font-sans text-base leading-none tabular-nums',
        'outline-none',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @change="onChange"
  />
</template>

<style scoped>
/*
 * The clock the machine draws inside the field. It comes with a spinner and a
 * cross of its own, and with a line box set from the browser's own chrome, and
 * the field is one row tall with the hour centred in it.
 */
.time-field::-webkit-inner-spin-button,
.time-field::-webkit-clear-button,
.time-field::-webkit-calendar-picker-indicator {
  display: none;
  -webkit-appearance: none;
  appearance: none;
}

.time-field::-webkit-datetime-edit {
  padding: 0;
  line-height: 1;
}

.time-field::-webkit-datetime-edit-fields-wrapper {
  padding: 0;
}
</style>
