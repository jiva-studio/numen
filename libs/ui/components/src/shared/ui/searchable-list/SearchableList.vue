<script setup lang="ts">
/**
 * One search line over a list of rows, in one frame: the way every list that
 * is searched is drawn.
 *
 * The keyboard stays on the search line and the arrows move the active row,
 * which Enter chooses. Rows may be set under group names, and an action row may
 * stand first. A press outside it and Escape ask for it to be closed.
 */
import { computed, provide, useId, useTemplateRef, watch } from 'vue'
import { useOutsidePress } from '../menu'
import SearchableListRows from './SearchableListRows.vue'
import type { SearchableListProps, SearchableListSlots } from './SearchableList.types'
import { LIST_CONTEXT } from './context'
import { useListFocus } from './useListFocus'
import { useListRows } from './useListRows'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<SearchableListProps>(), {
  placeholder: 'Search',
  emptyText: 'Nothing found',
  pendingText: 'Searching',
  maxRows: 8,
  isFiltered: true,
  isPending: false,
  hasAction: false,
  needsQuery: false,
})

/** What is typed on the search line. */
const query = defineModel<string>('query', { default: '' })

/** The item in force, by the caller's identifier. */
const model = defineModel<string | null>({ default: null })

const emit = defineEmits<{
  /** An item was chosen. */
  (event: 'select', id: string): void
  /** The action row was chosen, with the query as written. */
  (event: 'act', query: string): void
  /** The list asks to be put away: Escape, or a press outside it. */
  (event: 'close'): void
}>()

const slots = defineSlots<SearchableListSlots>()

/* --------------------------------- State ---------------------------------- */
const frame = useTemplateRef<HTMLElement>('frame')
const input = useTemplateRef<HTMLInputElement>('input')
const rowList = useTemplateRef<InstanceType<typeof SearchableListRows>>('rowList')

const listId = useId()

const { rows, queryText, isListOpen, activeKey, activeIndex, moveBy, moveToEnd, findRow } =
  useListRows({
    items: () => props.items,
    query,
    current: () => currentIds.value[0] ?? null,
    isFiltered: () => props.isFiltered,
    hasAction: () => props.hasAction,
    needsQuery: () => props.needsQuery,
  })

/** The items in force: the chosen ones where the caller gave them, else the model. */
const currentIds = computed<readonly string[]>(() => {
  if (props.chosen) return props.chosen
  return model.value === null ? [] : [model.value]
})

provide(LIST_CONTEXT, { slots, getQuery: () => queryText.value })

/* --------------------------------- Hooks ---------------------------------- */
useOutsidePress(frame, () => emit('close'))

useListFocus({ input, rowList, activeIndex, model })

// A new item in force set from outside puts the keyboard back on it, over a row the pointer rests on.
// A row chosen in the list is already the active row and stays it.
watch(model, (next) => {
  const active = activeKey.value === null ? undefined : findRow(activeKey.value)
  if (active?.kind !== 'item' || active.item.id !== next) activeKey.value = null
})

/* -------------------------------- Handlers -------------------------------- */
function onKey(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    event.stopPropagation()
    event.preventDefault()
    emit('close')
  } else if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    moveBy(event.key === 'ArrowDown' ? 1 : -1)
  } else if (event.key === 'Home' || event.key === 'End') {
    // The caret goes to the ends of what is written; with nothing written the rows do.
    if (query.value !== '') return
    event.preventDefault()
    moveToEnd(event.key === 'Home')
  } else if (event.key === 'Enter') {
    event.preventDefault()
    const row = rows.value[activeIndex.value]
    if (row) onChoose(row.key)
  }
}

function onChoose(key: string): void {
  const row = findRow(key)
  if (row?.kind === 'action') {
    emit('act', queryText.value)
  } else if (row && row.item.isDisabled !== true) {
    model.value = row.item.id
    emit('select', row.item.id)
  }
}
</script>

<template>
  <div
    ref="frame"
    class="searchable-list numen"
    data-numen-searchable-list
    role="group"
    :aria-label="label"
    :aria-busy="isPending"
    :style="{ '--rows': maxRows }"
  >
    <input
      ref="input"
      v-model="query"
      class="searchable-list__input"
      type="text"
      role="combobox"
      autocomplete="off"
      spellcheck="false"
      aria-autocomplete="list"
      :aria-label="placeholder"
      :aria-expanded="rows.length > 0"
      :aria-controls="listId"
      :aria-activedescendant="rows.length > 0 ? `${listId}-${activeIndex}` : undefined"
      :placeholder="placeholder"
      @keydown="onKey"
    />

    <div class="searchable-list__area" :data-open="isListOpen">
      <div class="searchable-list__clip">
        <SearchableListRows
          v-if="isListOpen && rows.length > 0"
          ref="rowList"
          :rows="rows"
          :active-index="activeIndex"
          :current="currentIds"
          :list-id="listId"
          :label="label"
          @choose="onChoose"
          @hover="activeKey = $event"
        />
        <p v-else-if="isListOpen" class="searchable-list__status" role="status">
          {{ isPending ? pendingText : emptyText }}
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* One frame: the search line, a rule, and the rows, drawn as a menu is. The rows
   open and close by their height. */
.searchable-list {
  /* The padding of the rows, and the height of one of them, which the visible
     number of rows is counted in. */
  --inset: 0.375rem;
  --row: calc(var(--numen-font-size) * var(--numen-line-height) + 0.75rem);
  /* The air at the sides of the search line and of the status line under it. */
  --side: 0.75rem;

  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
  color: var(--numen-ink);
  background-color: var(--numen-raised);
  border: var(--numen-stroke) solid var(--numen-panel-border);
  border-radius: var(--numen-radius-popup);
  box-shadow: var(--numen-panel-shadow);
}

.searchable-list__input {
  box-sizing: border-box;
  flex: none;
  inline-size: 100%;
  block-size: var(--numen-action-size);
  padding-block: 0;
  padding-inline: var(--side);
  font: inherit;
  color: var(--numen-ink);
  background: transparent;
  border: none;
  border-radius: 0;
  outline: none;
  box-shadow: none;
}

.searchable-list__input::placeholder {
  color: var(--numen-hushed);
}

/* The rows open and close by their height. */
.searchable-list__area {
  display: grid;
  grid-template-rows: 0fr;
  min-block-size: 0;
  transition: grid-template-rows var(--numen-motion) ease;
}

.searchable-list__area[data-open='true'] {
  grid-template-rows: 1fr;
}

@media (prefers-reduced-motion: reduce) {
  .searchable-list__area {
    transition: none;
  }
}

.searchable-list__clip {
  min-block-size: 0;
  overflow: hidden;
}

.searchable-list__status {
  margin: 0;
  padding: var(--inset) var(--side);
  color: var(--numen-hushed);
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}
</style>
