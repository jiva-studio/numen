<script setup lang="ts">
/**
 * Several choices out of a list, each switched on or off from a searchable list that stays open.
 * The line says the first choice in force and how many more. A choice pressed emits `toggle` with
 * its identifier and whether it is now in force, and `update:selected` with the new list.
 */
import { computed, useTemplateRef } from 'vue'
import { SearchableListPopup } from '@numen/ui'
import { useMultiSelect } from '../../model/useMultiSelect'
import { useMultiSelectText } from '../../model/useMultiSelectText'
import type { MultiSelectProps } from './MultiSelect.types'
import SelectTrigger from './SelectTrigger.vue'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<MultiSelectProps>(), {
  placeholder: '',
  name: 'Choices',
  isDisabled: false,
  searchName: 'Search',
  getMoreText: (count: number) => `+${count}`,
})

const emit = defineEmits<{
  (event: 'update:selected', selected: readonly string[]): void
  (event: 'toggle', id: string, isOn: boolean): void
}>()

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  get element() {
    return element.value
  },
  focus: (how?: FocusOptions) => element.value?.focus(how),
})

/* ---------------------------------- State --------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof SelectTrigger>>('trigger')
const element = computed(() => trigger.value?.element ?? null)
const { query, menuAnchor, openChoices, closeChoices, toggleChoice } = useMultiSelect({
  getSelected: () => props.selected,
  isDisabled: () => props.isDisabled,
  getElement: () => element.value,
  updateSelected: (selected) => emit('update:selected', selected),
  toggle: (id, isOn) => emit('toggle', id, isOn),
})

const displayText = useMultiSelectText(() => props)

/* -------------------------------- Handlers -------------------------------- */
const onKey = (event: KeyboardEvent) => {
  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
  event.preventDefault()
  openChoices()
}
</script>

<template>
  <SelectTrigger
    ref="trigger"
    v-bind="$attrs"
    :text="displayText"
    :is-hushed="selected.length === 0"
    :is-disabled="isDisabled"
    :is-expanded="menuAnchor !== null"
    has-popup="listbox"
    :class="props.class"
    @click="openChoices"
    @keydown="onKey"
  />

  <SearchableListPopup
    v-if="menuAnchor"
    v-model:query="query"
    :model-value="null"
    :chosen="selected"
    :items="choices"
    :label="name"
    :placeholder="searchName"
    empty-text="Nothing to choose"
    :at="menuAnchor.at"
    :asking="menuAnchor.wide"
    :from="element"
    @select="toggleChoice"
    @close="closeChoices"
  />
</template>
