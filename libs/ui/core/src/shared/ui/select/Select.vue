<script setup lang="ts">
/**
 * One choice out of a list, selected from the product's menu. Choices that name
 * a group are shown under the group name.
 *
 * A current value that matches none of the choices is displayed as it is, and
 * no event is emitted until a choice is made. A searchable select offers the list
 * with a search line over it, and may take what is typed as the value.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { measureElementRect } from '@/shared/lib/dom'
import { cn } from '@/shared/lib/dom/classes'
import { Menu, type MenuItem } from '../menu'
import { SearchableListPopup, filterItems } from '../searchable-list'
import type { Position } from '@/shared/lib/math/geometry'
import type { SelectProps } from './Select.types'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<SelectProps>(), {
  placeholder: '',
  name: 'Choices',
  isDisabled: false,
  isSearchable: false,
  canEnterCustom: false,
  isMultiple: false,
  searchName: 'Search',
})

/** Which choice is in force, by the identifier the caller gave it. */
const model = defineModel<string>({ default: '' })

/** Which choices are in force in a multiple select, by identifier. */
const selected = defineModel<readonly string[]>('selected', { default: () => [] })

/* ---------------------------------- State --------------------------------- */
const items = computed<readonly MenuItem[]>(() =>
  props.choices.map((one) => ({
    id: one.id,
    text: one.text,
    ...(one.detail ? { detail: one.detail } : {}),
    ...(one.group ? { group: one.group } : {}),
  })),
)

/** What is typed on the search line of a searchable select. */
const query = ref('')

/** Whether what is typed is offered as the value: it names no choice and is not empty. */
const isCustomAvailable = computed(
  () =>
    props.canEnterCustom &&
    !props.isMultiple &&
    query.value.trim() !== '' &&
    filterItems(props.choices, query.value).length === 0,
)

/** The choice in force, and none where the value is none of them. */
const selectedChoice = computed(() => props.choices.find((one) => one.id === model.value))

const getChosenText = (id: string): string => props.choices.find((one) => one.id === id)?.text || id

/** What is written on the line of a multiple select: the first choice, and how many more. */
const multipleText = computed(() => {
  const [first, ...rest] = selected.value
  if (first === undefined) return props.placeholder
  return rest.length === 0 ? getChosenText(first) : `${getChosenText(first)} +${rest.length}`
})

const displayText = computed(() =>
  props.isMultiple
    ? multipleText.value
    : selectedChoice.value?.text || model.value || props.placeholder,
)

const isEmpty = computed(() => (props.isMultiple ? selected.value.length === 0 : !model.value))

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
const onOpenChoices = () => {
  const line = element.value
  if (!line || props.isDisabled) return
  const box = measureElementRect(line)
  menuAnchor.value = { at: { x: box.left, y: box.bottom }, wide: box.width }
}

const onCloseChoices = () => {
  menuAnchor.value = null
  query.value = ''
}

const onChoose = (id: string) => {
  if (props.isMultiple) {
    selected.value = selected.value.includes(id)
      ? selected.value.filter((one) => one !== id)
      : [...selected.value, id]
    return
  }
  model.value = id
  onCloseChoices()
}

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
    :aria-haspopup="isSearchable ? 'listbox' : 'menu'"
    :aria-expanded="menuAnchor !== null"
    :disabled="isDisabled"
    :class="
      cn(
        'flex w-full max-w-full min-w-0 items-center justify-between gap-2',
        'h-action rounded-node border-field-rule bg-field border px-2.5',
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
      :class="{ 'text-hushed': isEmpty && !selectedChoice }"
      :title="displayText"
    >
      {{ displayText }}
    </span>
    <ChevronDown class="select__mark shrink-0" aria-hidden="true" />
  </button>

  <SearchableListPopup
    v-if="menuAnchor && (isSearchable || isMultiple)"
    v-model:query="query"
    :model-value="isMultiple ? null : model"
    :chosen="isMultiple ? selected : undefined"
    :items="choices"
    :label="name"
    :placeholder="searchName"
    empty-text="Nothing to choose"
    :has-action="isCustomAvailable"
    :at="menuAnchor.at"
    :asking="menuAnchor.wide"
    :from="element"
    @select="onChoose"
    @act="onChoose"
    @close="onCloseChoices"
  >
    <template #action="{ query: text }">
      <span class="select__custom">{{ text }}</span>
      <span class="select__custom-note">Use custom value</span>
    </template>
  </SearchableListPopup>

  <Menu
    v-else-if="menuAnchor"
    :items="items"
    :at="menuAnchor.at"
    :asking="menuAnchor.wide"
    :from="element"
    :current="model"
    :name="name"
    has-groups
    is-open
    trigger="keyboard"
    @choose="onChoose"
    @dismiss="menuAnchor = null"
  >
    <template #silence>Nothing to choose</template>
  </Menu>
</template>

<style scoped>
/* The typed value, and the note that it is not one of the choices. */
.select__custom-note {
  margin-inline-start: auto;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

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
