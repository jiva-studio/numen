<script setup lang="ts">
/**
 * A searchable list standing over the page at a point, as a menu does: kept
 * inside the window, put away by a press outside it, a scroll, a resize and
 * Escape, and giving the keyboard back to what asked for it.
 */
import { computed, onBeforeUnmount, shallowRef, useTemplateRef } from 'vue'
import { measureElementRect } from '@/shared/api/dom'
import { useMenuDismissal, useMenuPlacement } from '../menu'
import SearchableList from './SearchableList.vue'
import { useListMeasure } from './useListMeasure'
import type {
  SearchableListPopupProps,
  SearchableListProps,
  SearchableListSlots,
} from './SearchableList.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<SearchableListProps & SearchableListPopupProps>(), {
  placeholder: 'Search',
  emptyText: 'Nothing found',
  pendingText: 'Searching',
  maxRows: 8,
  isFiltered: true,
  isPending: false,
  hasAction: false,
  needsQuery: false,
  asking: 0,
  viewport: null,
  from: null,
})

const query = defineModel<string>('query', { default: '' })

const model = defineModel<string | null>({ default: null })

const emit = defineEmits<{
  (event: 'select', id: string): void
  (event: 'act', query: string): void
  (event: 'close'): void
}>()

defineSlots<SearchableListSlots>()

/* --------------------------------- State ---------------------------------- */
const list = useTemplateRef<InstanceType<typeof SearchableList>>('list')
const frame = shallowRef<HTMLElement | null>(null)

/** What the list itself takes: everything but where it is placed. */
const listProps = computed(() => {
  const { at: _at, asking: _asking, viewport: _viewport, from: _from, ...own } = props
  return own
})

const { setSize, menuStyle } = useMenuPlacement({
  at: () => props.at,
  viewport: () => props.viewport,
  margin: () => 8,
  asking: () => props.asking,
})

/* --------------------------------- Hooks ---------------------------------- */
const dismissal = useMenuDismissal(frame, () => emit('close'))

useListMeasure({
  mount: () => {
    frame.value = list.value?.$el ?? null
    dismissal.listen()
  },
  getChangeKey: () => [props.items, props.at, query.value],
  measure,
})

onBeforeUnmount(() => {
  if (props.from?.isConnected) props.from.focus()
})

/* -------------------------------- Helpers --------------------------------- */
function measure(): void {
  const box = frame.value ? measureElementRect(frame.value) : undefined
  if (box) setSize({ width: box.width, height: box.height })
}
</script>

<template>
  <Teleport to="body">
    <SearchableList
      ref="list"
      v-bind="listProps"
      v-model="model"
      v-model:query="query"
      class="searchable-list-popup"
      :style="menuStyle"
      @select="emit('select', $event)"
      @act="emit('act', $event)"
      @close="emit('close')"
    >
      <template #leading="{ item }"><slot name="leading" :item="item" /></template>
      <template #trailing="{ item }"><slot name="trailing" :item="item" /></template>
      <template #action="{ query: text }"><slot name="action" :query="text" /></template>
    </SearchableList>
  </Teleport>
</template>

<style scoped>
/* Over the page at the point the placement worked out, at least as wide as what
   asked for it. */
.searchable-list-popup {
  --narrowest: 11.25rem;
  --widest: 22rem;
  --asking: 0px;
  --opened-width: 0px;
  --room: calc(100dvh - 1rem);

  position: fixed;
  z-index: var(--numen-lift-menu);
  inline-size: max-content;
  min-inline-size: max(var(--narrowest), var(--asking), var(--opened-width));
  max-inline-size: max(var(--widest), var(--asking));
  max-block-size: var(--room);
}
</style>
