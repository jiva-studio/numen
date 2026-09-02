<script setup lang="ts">
/**
 * An hour and a minute of the day, written as `04:00` and typed on the clock
 * the machine draws.
 *
 * A field standing at an hour of the day hands that hour on. A field standing
 * at half an hour, or at nothing, hands nothing on and is left where it stands.
 */
import { computed, useTemplateRef, type HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
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
const standing = computed(() => (onTheClock(model.value) ? model.value : ''))

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
    :value="standing"
    :disabled="disabled"
    :min="min || undefined"
    :max="max || undefined"
    :class="
      cn(
        'w-full rounded-tight border border-field-rule bg-field',
        // The gaps are measured to the ink the screen paints, which is what a
        // field of typing beside it is measured to.
        'px-2 py-1.5',
        'font-sans text-base leading-none text-ink tabular-nums',
        'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @change="took"
  />
</template>
