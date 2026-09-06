<script setup lang="ts">
/**
 * An hour and a minute of the day, written as `04:00` and typed on the clock
 * the machine draws.
 *
 * A field standing at an hour of the day hands that hour on. A field standing
 * at half an hour, or at nothing, hands nothing on and is left where it stands.
 */
import { computed, useTemplateRef, type HTMLAttributes } from 'vue'
import { cn } from '@/classes'
import { onTheClock } from './clock'

const props = withDefaults(
  defineProps<{
    /** How early and how late in the day the hour may stand, written as `04:00`. */
    min?: string
    max?: string
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { min: '', max: '', disabled: false },
)

/** The hour in force, written as `04:00`. An empty field holds none. */
const model = defineModel<string>({ default: '' })

const raises = defineEmits<{
  /** The field come to rest at an hour of the day. */
  settles: [value: string]
}>()

/** What stands in the field, which is an hour of the day or nothing at all. */
const inForce = computed(() => (onTheClock(model.value) ? model.value : ''))

const element = useTemplateRef<HTMLInputElement>('element')

const took = (event: Event) => {
  const said = (event.target as HTMLInputElement).value
  if (!onTheClock(said) || said === model.value) return
  model.value = said
  raises('settles', said)
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
    type="time"
    data-slot="time-field"
    :value="inForce"
    :disabled="disabled"
    :min="min || undefined"
    :max="max || undefined"
    :class="
      cn(
        'time-field w-full rounded-tight border border-field-rule bg-field',
        // One row tall, which every control standing on a row is drawn at.
        'h-action px-2',
        'font-sans text-base leading-none text-ink tabular-nums',
        'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @change="took"
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
