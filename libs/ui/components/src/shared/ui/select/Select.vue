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
import { measureElementRect } from '@/shared/api/dom'
import { Menu, type MenuItem } from '../menu'
import { SearchableListPopup, filterItems } from '../searchable-list'
import type { Position } from '@/shared/lib/math/geometry'
import type { SelectProps } from './Select.types'
import SelectTrigger from './SelectTrigger.vue'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<SelectProps>(), {
  placeholder: '',
  name: 'Choices',
  isDisabled: false,
  isSearchable: false,
  canEnterCustom: false,
  searchName: 'Search',
})

/** Which choice is in force, by the identifier the caller gave it. */
const model = defineModel<string>({ default: '' })

defineExpose({
  /** The element itself, for a caller that has to reach past these props. */
  get element() {
    return element.value
  },
  focus: (how?: FocusOptions) => element.value?.focus(how),
})

/* ---------------------------------- State --------------------------------- */
const items = computed<readonly MenuItem[]>(() =>
  props.choices.map((one) => ({
    id: one.id,
    text: one.text,
    ...(one.detail ? { detail: one.detail } : {}),
    ...(one.group ? { group: one.group } : {}),
    ...(one.isDisabled ? { isDisabled: true } : {}),
  })),
)

/** What is typed on the search line of a searchable select. */
const query = ref('')

/** Whether what is typed is offered as the value: it names no choice and is not empty. */
const isCustomAvailable = computed(
  () =>
    props.canEnterCustom &&
    query.value.trim() !== '' &&
    filterItems(props.choices, query.value).length === 0,
)

/** The choice in force, and none where the value is none of them. */
const selectedChoice = computed(() => props.choices.find((one) => one.id === model.value))

const displayText = computed(() => selectedChoice.value?.text || model.value || props.placeholder)

const trigger = useTemplateRef<InstanceType<typeof SelectTrigger>>('trigger')
const element = computed(() => trigger.value?.element ?? null)

/**
 * Where the choices are drawn and how wide the line asking for them is, and
 * nothing while they are not drawn at all.
 */
const menuAnchor = ref<{ at: Position; wide: number } | null>(null)

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
  <SelectTrigger
    ref="trigger"
    v-bind="$attrs"
    :text="displayText"
    :is-hushed="!model && !selectedChoice"
    :is-disabled="isDisabled"
    :is-expanded="menuAnchor !== null"
    :has-popup="isSearchable ? 'listbox' : 'menu'"
    :class="props.class"
    @click="onOpenChoices"
    @keydown="onKey"
  />

  <SearchableListPopup
    v-if="menuAnchor && isSearchable"
    v-model:query="query"
    :model-value="model"
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
</style>
