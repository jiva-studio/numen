<script setup lang="ts">
/**
 * The card of the connections page: a row for each connection and the Add button below them.
 */
import { useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import type { ConnectionInfo, ConnectionTypeInfo } from '@/entities/connection'
import { WORDS as words } from '../../words'
import ConnectionRow from './ConnectionRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  connections: readonly ConnectionInfo[]
  types: readonly ConnectionTypeInfo[]
  hasType: boolean
}>()

const emit = defineEmits<{
  (e: 'edit', connection: ConnectionInfo, opener: HTMLElement): void
  (e: 'add'): void
}>()

defineExpose({ focusAdd, focusRow })

/* --------------------------------- State ---------------------------------- */
const addButton = useTemplateRef<HTMLButtonElement>('addButton')
const list = useTemplateRef<HTMLElement>('list')

/* -------------------------------- Handlers -------------------------------- */
function onAdd(): void {
  emit('add')
}

/* -------------------------------- Helpers --------------------------------- */
function getType(connection: ConnectionInfo): ConnectionTypeInfo | undefined {
  return props.types.find((one) => one.id === connection.typeId)
}

function focusAdd(): void {
  addButton.value?.focus()
}

function focusRow(at: number): void {
  const rows = Array.from(list.value?.children ?? [])
  const element = rows[at] ?? rows[at - 1] ?? addButton.value
  if (element instanceof HTMLElement) element.focus()
}
</script>

<template>
  <div class="settings-card connection-list">
    <div ref="list" role="list" class="connection-list__rows">
      <ConnectionRow
        v-for="connection in connections"
        :key="connection.id"
        :connection="connection"
        :type="getType(connection)"
        @edit="(opener) => emit('edit', connection, opener)"
      />
    </div>
    <button
      ref="addButton"
      type="button"
      class="connection-list__add"
      :disabled="!hasType"
      @click="onAdd"
    >
      <Plus class="connection-list__add-glyph" :size="20" :stroke-width="1.75" aria-hidden="true" />
      {{ words.connectionAdd }}
    </button>
  </div>
</template>

<style scoped>
.connection-list {
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
}

.connection-list__rows {
  display: flex;
  flex-direction: column;
}

.connection-list__add {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  box-sizing: border-box;
  inline-size: 100%;
  min-block-size: var(--settings-row-min);
  padding: var(--settings-row-pad) var(--settings-card-pad);
  border: none;
  background: none;
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
  font-weight: 500;
  text-align: start;
  cursor: pointer;
  transition: background-color var(--numen-motion-hover) ease;
}

.connection-list__rows:not(:empty) + .connection-list__add {
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
}

.connection-list__add:hover:not(:disabled) {
  background-color: var(--settings-hover-bg);
}

.connection-list__add:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: calc(-1 * var(--numen-stroke));
}

.connection-list__add:disabled {
  color: var(--numen-hushed);
  cursor: default;
}

.connection-list__add-glyph {
  flex: none;
  color: var(--numen-hushed);
}
</style>
