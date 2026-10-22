<script setup lang="ts">
/**
 * One choice out of a list, taken from the menu this product draws. Choices
 * naming a group are drawn under its name.
 *
 * A value in force that is none of the choices is written on the line as it
 * stands, and the caller is told nothing until a choice is made.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { cn } from '@/shared/lib/classes'
import { Menu, type MenuItem } from '../menu'
import type { Position } from '@/shared/lib/geometry'
import type { SelectProps } from './Select.types'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<SelectProps>(), {
  placeholder: '',
  name: 'Choices',
  isDisabled: false,
  isSearchable: false,
  searchName: 'Search',
})

/** Which choice is in force, by the identifier the caller gave it. */
const model = defineModel<string>({ default: '' })

/* ---------------------------------- State --------------------------------- */
const items = computed<readonly MenuItem[]>(() =>
  props.choices.map((one) => ({
    id: one.id,
    text: one.text,
    ...(one.detail ? { detail: one.detail } : {}),
    ...(one.group ? { group: one.group } : {}),
  })),
)

/** The choice in force, and none where the value is none of them. */
const selectedChoice = computed(() => props.choices.find((one) => one.id === model.value))

/** What is written on the line: the choice, the value itself, or the stand-in. */
const displayText = computed(() => selectedChoice.value?.text || model.value || props.placeholder)

const element = useTemplateRef<HTMLButtonElement>('element')

/**
 * Where the choices are drawn and how wide the line asking for them is, and
 * nothing while they are not drawn at all.
 */
const menuAnchor = ref<{ at: Position; wide: number } | null>(null)

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  element,
  focus: (how?: FocusOptions) => element.value?.focus(how),
})

/* -------------------------------- Handlers -------------------------------- */
/**
 * The line opens the choices under itself, along its own leading edge and no
 * narrower than itself.
 */
const onOpenChoices = () => {
  const line = element.value
  if (!line || props.isDisabled) return
  const box = line.getBoundingClientRect()
  menuAnchor.value = { at: { x: box.left, y: box.bottom }, wide: box.width }
}

const onChoose = (id: string) => {
  model.value = id
}

/** The arrows open the choices standing on the one in force. */
const onKey = (event: KeyboardEvent) => {
  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
  event.preventDefault()
  onOpenChoices()
}
</script>

<template>
  <button
    ref="element"
    v-bind="$attrs"
    type="button"
    data-slot="select"
    aria-haspopup="menu"
    :aria-expanded="menuAnchor !== null"
    :disabled="isDisabled"
    :class="
      cn(
        'flex w-full max-w-full min-w-0 items-center justify-between gap-2',
        'h-action rounded-tight border-field-rule bg-field border px-2',
        'text-ink text-left font-sans text-base leading-none',
        'ease-numen cursor-pointer transition-colors duration-100 outline-none',
        'hover:border-rule',
        'focus-visible:outline-none',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    @click="onOpenChoices"
    @keydown="onKey"
  >
    <span
      class="select__reading min-w-0"
      :class="{ 'text-hushed': !selectedChoice && !model }"
      :title="displayText"
    >
      {{ displayText }}
    </span>
    <ChevronDown class="select__mark shrink-0" aria-hidden="true" />
  </button>

  <Menu
    v-if="menuAnchor"
    :items="items"
    :at="menuAnchor.at"
    :asking="menuAnchor.wide"
    :from="element"
    :current="model"
    :name="name"
    has-groups
    :is-searchable="isSearchable"
    :search-name="searchName"
    is-open
    trigger="keyboard"
    @choose="onChoose"
    @dismiss="menuAnchor = null"
  >
    <template #silence>Nothing to choose</template>
  </Menu>
</template>

<style scoped>
/* One line, then an ellipsis. A model is named by a word and addressed by a
   line too long to read, and the name is what the line carries. */
.select__reading {
  padding-block: 0.125rem;
  line-height: 1.4;
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
