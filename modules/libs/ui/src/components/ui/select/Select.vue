<script setup lang="ts">
/**
 * One choice out of a list, taken from the menu this product draws. Choices
 * naming a shelf are drawn under its name.
 *
 * A value in force that is none of the choices is written on the line as it
 * stands, and the caller is told nothing until a choice is made.
 */
import { computed, ref, useTemplateRef, type HTMLAttributes } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { cn } from '@/classes'
import Menu from '../../../menu/Menu.vue'
import type { MenuItem } from '../../../menu/item'
import type { Point } from '../../../lib/geometry'
import type { SelectChoice } from '.'

const props = withDefaults(
  defineProps<{
    /** The choices, in the order they are offered. */
    choices: readonly SelectChoice[]
    /** What stands on the line while nothing is in force. */
    placeholder?: string
    /** What the list of choices is announced as. */
    name?: string
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { placeholder: '', name: 'Choices', disabled: false },
)

defineOptions({ inheritAttrs: false })

/** Which choice is in force, by the identifier the caller gave it. */
const model = defineModel<string>({ default: '' })

const items = computed<readonly MenuItem[]>(() =>
  props.choices.map((one) => ({
    id: one.id,
    text: one.text,
    ...(one.detail ? { detail: one.detail } : {}),
    ...(one.group ? { band: one.group } : {}),
  })),
)

/** The choice in force, and none where the value is none of them. */
const chosen = computed(() => props.choices.find((one) => one.id === model.value))

/** What is written on the line: the choice, the value itself, or the stand-in. */
const reading = computed(() => chosen.value?.text || model.value || props.placeholder)

const element = useTemplateRef<HTMLButtonElement>('element')

/**
 * Where the choices are drawn and how wide the line asking for them is, and
 * nothing while they are not drawn at all.
 */
const asking = ref<{ at: Point; wide: number } | null>(null)

/**
 * The line opens the choices under itself, along its own leading edge and no
 * narrower than itself.
 */
const opens = () => {
  const line = element.value
  if (!line || props.disabled) return
  const box = line.getBoundingClientRect()
  asking.value = { at: { x: box.left, y: box.bottom }, wide: box.width }
}

const chose = (id: string) => {
  model.value = id
}

/** The arrows open the choices standing on the one in force. */
const onKey = (event: KeyboardEvent) => {
  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
  event.preventDefault()
  opens()
}

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  element,
  focus: (how?: FocusOptions) => element.value?.focus(how),
})
</script>

<template>
  <button
    ref="element"
    v-bind="$attrs"
    type="button"
    data-slot="select"
    aria-haspopup="menu"
    :aria-expanded="asking !== null"
    :disabled="disabled"
    :class="
      cn(
        'flex w-full items-center justify-between gap-2',
        'h-action rounded-tight border border-field-rule bg-field px-2',
        'font-sans text-base leading-none text-ink text-left',
        'cursor-pointer outline-none transition-colors duration-100 ease-numen',
        'hover:border-rule',
        'focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @click="opens"
    @keydown="onKey"
  >
    <span class="select__reading min-w-0" :class="{ 'text-hushed': !chosen && !model }">
      {{ reading }}
    </span>
    <ChevronDown class="select__mark shrink-0" aria-hidden="true" />
  </button>

  <Menu
    v-if="asking"
    :items="items"
    :at="asking.at"
    :asking="asking.wide"
    :from="element"
    :current="model"
    :name="name"
    bands
    open
    opening="keyboard"
    @choose="chose"
    @dismiss="asking = null"
  >
    <template #silence>Nothing to choose</template>
  </Menu>
</template>

<style scoped>
/* One line, then an ellipsis. A model is named by a word and addressed by a
   line too long to read, and the name is what the line carries. */
.select__reading {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.select__mark {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
  color: var(--numen-hushed);
}
</style>
