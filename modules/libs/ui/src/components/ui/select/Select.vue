<script setup lang="ts">
/**
 * One choice out of a list, taken from a menu the machine draws.
 *
 * It stands as tall as a line of typing does, so a row of controls shares one
 * middle. Choices naming a shelf are drawn under it, in the runs they arrive in.
 *
 * A value in force that is none of the choices leaves the menu standing on
 * nothing, and the caller is told nothing until a choice is made.
 */
import { computed, useTemplateRef, type HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import { shelved } from './shelves'
import type { SelectChoice } from '.'

const props = withDefaults(
  defineProps<{
    /** The choices, in the order they are offered. */
    choices: readonly SelectChoice[]
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { disabled: false },
)

/** Which choice is in force, by the identifier the caller gave it. */
const model = defineModel<string>({ default: '' })

const shelves = computed(() => shelved(props.choices))

const element = useTemplateRef<HTMLSelectElement>('element')

const chose = (event: Event) => {
  model.value = (event.target as HTMLSelectElement).value
}

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  element,
  focus: (how?: FocusOptions) => element.value?.focus(how),
})
</script>

<template>
  <select
    ref="element"
    data-slot="select"
    :value="model"
    :disabled="disabled"
    :class="
      cn(
        'w-full rounded-tight border border-field-rule bg-field',
        // The gaps are measured to the ink the screen paints, which is what a
        // field of typing beside it is measured to.
        'px-2 py-1.5',
        'font-sans text-base leading-none text-ink',
        'cursor-pointer outline-none',
        'focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @change="chose"
  >
    <template v-for="(shelf, at) in shelves" :key="shelf.label ?? at">
      <optgroup v-if="shelf.label" :label="shelf.label">
        <option v-for="one in shelf.choices" :key="one.id" :value="one.id">{{ one.text }}</option>
      </optgroup>
      <template v-else>
        <option v-for="one in shelf.choices" :key="one.id" :value="one.id">{{ one.text }}</option>
      </template>
    </template>
  </select>
</template>
