<script setup lang="ts">
/**
 * One row of a searchable list: the words of an item with what a caller draws
 * before and after them, or the action row. The slots are the caller's, read from
 * the list that holds the row.
 */
import { inject } from 'vue'
import { Check } from '@lucide/vue'
import { LIST_CONTEXT } from './context'
import type { ListRow } from './rows'

defineProps<{
  row: ListRow
  id: string
  heading?: string | undefined
  hasSeparator?: boolean
  isActive: boolean
  isCurrent: boolean
}>()

defineEmits<{
  /** The row was pressed. */
  (event: 'choose'): void
  /** The pointer moved onto the row. */
  (event: 'hover'): void
}>()

const context = inject(LIST_CONTEXT)
</script>

<template>
  <hr v-if="hasSeparator" class="searchable-list__separator" role="separator" />
  <p v-if="heading !== undefined" class="searchable-list__heading" role="presentation">
    {{ heading }}
  </p>
  <div
    :id="id"
    class="searchable-list__option"
    role="option"
    :aria-selected="isActive"
    :aria-current="isCurrent ? 'true' : undefined"
    @mousedown.prevent
    @click="$emit('choose')"
    @pointerenter="$emit('hover')"
  >
    <component
      :is="context?.slots.action"
      v-if="row.kind === 'action'"
      :query="context?.getQuery() ?? ''"
    />
    <template v-else>
      <component :is="context?.slots.leading" :item="row.item" />
      <span class="searchable-list__words">
        <span class="searchable-list__text">{{ row.item.text }}</span>
        <span v-if="row.item.detail" class="searchable-list__detail">{{ row.item.detail }}</span>
      </span>
      <component :is="context?.slots.trailing" :item="row.item" />
      <Check v-if="isCurrent" class="searchable-list__mark" aria-hidden="true" />
    </template>
  </div>
</template>

<style scoped>
/* The line above an item that is set apart from the rows before it. */
.searchable-list__separator {
  /* The space the line keeps on each side, as a rule of a menu keeps. */
  --parting: 0.25rem;

  block-size: 0;
  margin-block: var(--parting);
  border: 0;
  border-block-start: 1px solid var(--numen-panel-border);
}

/* The name of a group, set as this product sets a label over what it names. */
.searchable-list__heading {
  --pad: 0.25rem 0.5rem;

  margin: 0;
  padding: var(--pad);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
  color: var(--numen-hushed);
}

.searchable-list__option {
  --gap: 0.5rem;
  --pad: 0.375rem 0.5rem;

  box-sizing: border-box;
  display: flex;
  align-items: center;
  gap: var(--gap);
  padding: var(--pad);
  border-radius: var(--numen-radius-popup-item);
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
}

.searchable-list__option[aria-selected='true'] {
  background: var(--numen-bubble-bg);
}

.searchable-list__words {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-inline-size: 0;
}

/* One line, then an ellipsis. A list is read down its leading edge. */
.searchable-list__text,
.searchable-list__detail {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.searchable-list__detail {
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

/* The hushed detail is too faint on the highlight; the row is read in the ink. */
.searchable-list__option[aria-selected='true'] .searchable-list__detail {
  color: var(--numen-ink);
}

.searchable-list__mark {
  --mark: 0.875rem;

  flex: none;
  inline-size: var(--mark);
  block-size: var(--mark);
}
</style>
