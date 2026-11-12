<script setup lang="ts">
/**
 * One row of a setting of rows: the grip that moves it, the connection, one control for each row field
 * of the type of the connection, such as a model and a reasoning level, and the button that removes it.
 * The values of a field are asked from the provider of the connection; a text field also takes a typed
 * value. Choosing a value clears the fields after it, whose values depend on it. The list of connections
 * ends with "New connection…", which opens the dialog that adds one; the saved connection is chosen in
 * the row and the list of its first field opens, so the person chooses it at once. Closing the dialog
 * leaves the row as it was.
 */
import { computed, nextTick, ref } from 'vue'
import { GripVertical, X } from '@lucide/vue'
import { Button, Select, runInBackground } from '@numen/ui'
import type { ConnectionInfo, StoredRow } from '@/entities/connection'
import { WORDS as words } from '../../../words'
import type { ConnectionRowEditorProps } from './ConnectionRowEditor.types'
import { openSelect } from '../../../model/openSelect'
import { useRowChoices } from '../../../model/useRowChoices'
import type { SelectChoice } from '../../../types'
import ConnectionDialog from '../../connections/ConnectionDialog.vue'

/** The identifier of the entry that adds a connection; no stored connection has it. */
const NEW_CONNECTION = 'new-connection'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<ConnectionRowEditorProps>()

const emit = defineEmits<{
  (e: 'update', patch: Partial<StoredRow>): void
  (e: 'remove'): void
  (e: 'grip-press', event: PointerEvent): void
  (e: 'grip-key', event: KeyboardEvent): void
}>()

/* --------------------------------- State ---------------------------------- */
const { fields } = useRowChoices({
  getRow: () => props.row,
  getRowFields: () => props.rowFields,
  getOperations: () => props.operations,
  getListRowOptions: () => props.listRowOptions,
  update: (options) => emit('update', { options }),
})

const isAdding = ref(false)
const fieldSelects = ref<InstanceType<typeof Select>[]>([])
const connectionChoices = computed<readonly SelectChoice[]>(() => [
  ...props.choices,
  { id: NEW_CONNECTION, text: words.connectionNew },
])

/* -------------------------------- Handlers -------------------------------- */
function onConnectionChoose(connection: string): void {
  if (connection === NEW_CONNECTION) {
    isAdding.value = true
    return
  }
  if (connection === props.row.connection) return
  emit('update', { connection, options: {} })
}

function onConnectionSaved(saved: ConnectionInfo): void {
  emit('update', { connection: saved.id, options: {} })
  runInBackground(
    nextTick(() => openSelect(fieldSelects.value[0]?.element)),
    () => undefined,
  )
}

function onFieldChoose(index: number, value: string): void {
  const field = props.rowFields[index]
  if (!field || value === (props.row.options[field.key] ?? '')) return
  const kept = props.rowFields
    .slice(0, index)
    .map((one) => [one.key, props.row.options[one.key] ?? ''])
  emit('update', {
    options: Object.fromEntries([...kept, [field.key, value]].filter(([, one]) => one !== '')),
  })
}

/* -------------------------------- Helpers --------------------------------- */
const columns = computed(() =>
  [
    'minmax(0, 2fr)',
    ...props.rowFields.map((one) => (one.isRequired ? 'minmax(0, 3fr)' : 'minmax(0, 1.2fr)')),
  ].join(' '),
)
</script>

<template>
  <li
    class="connection-rows__row"
    :class="{
      'connection-rows__row--held': isHeld,
      'connection-rows__row--dragging': isDragging,
      'connection-rows__row--target': isDropTarget,
    }"
    :data-row-key="row.key"
  >
    <button
      v-if="!isSingle"
      type="button"
      class="connection-rows__grip"
      :data-grip="row.key"
      :aria-label="words.connectionRowsGrip(position, count)"
      :aria-pressed="isHeld"
      @pointerdown="emit('grip-press', $event)"
      @keydown="emit('grip-key', $event)"
    >
      <GripVertical :size="14" aria-hidden="true" />
    </button>

    <div class="connection-rows__fields" :style="{ '--connection-rows-columns': columns }">
      <Select
        :model-value="row.connection"
        :choices="connectionChoices"
        :name="words.connectionRowsConnection"
        :placeholder="words.connectionRowsChoose"
        :aria-label="words.connectionRowsConnection"
        :search-name="words.searchConnections"
        :problem="reason"
        is-searchable
        class="connection-rows__select"
        @update:model-value="onConnectionChoose"
      />
      <Select
        v-for="(one, index) in fields"
        :key="one.field.key"
        ref="fieldSelects"
        :model-value="one.value"
        :choices="one.choices"
        :name="one.field.title"
        :aria-label="one.field.title"
        :problem="one.problem"
        :is-disabled="one.isDisabled"
        v-bind="
          one.field.type === 'string'
            ? {
                placeholder: words.connectionRowsValuePlaceholder(one.field.title),
                searchName: words.searchValues(one.field.title),
                isSearchable: true,
                canEnterCustom: true,
              }
            : {}
        "
        class="connection-rows__select"
        @update:model-value="(value: string) => onFieldChoose(index, value)"
      />
    </div>

    <Button
      v-if="!isSingle"
      variant="ghost"
      size="icon"
      class="connection-rows__remove"
      :aria-label="words.connectionRowsRemove(position)"
      :title="words.connectionRowsRemove(position)"
      @click="emit('remove')"
    >
      <X :size="14" aria-hidden="true" />
    </Button>

    <ConnectionDialog
      v-if="isAdding"
      :types="types"
      :create="create"
      :update="update"
      @saved="onConnectionSaved"
      @close="isAdding = false"
    />
  </li>
</template>

<style scoped>
.connection-rows__row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem 0.5rem;
  align-items: center;
  padding-block: 0.625rem;
  list-style: none;
}

.connection-rows__row + .connection-rows__row {
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
}

.connection-rows__row--dragging {
  opacity: 0.5;
}

.connection-rows__row--target {
  box-shadow: inset 0 var(--numen-stroke) 0 var(--numen-accent);
}

.connection-rows__row--held {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: calc(-1 * var(--numen-stroke));
}

.connection-rows__grip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  inline-size: 1.5rem;
  block-size: 2rem;
  padding: 0;
  border: none;
  border-radius: var(--numen-radius);
  background: none;
  color: var(--numen-hushed);
  cursor: grab;
  touch-action: none;
}

.connection-rows__grip:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.connection-rows__grip[aria-pressed='true'] {
  color: var(--numen-accent);
  cursor: grabbing;
}

.connection-rows__fields {
  display: grid;
  flex: 1 1 0;
  grid-template-columns: var(--connection-rows-columns);
  gap: 0.375rem;
  min-inline-size: 0;
}

.connection-rows__select {
  inline-size: 100%;
  min-inline-size: 0;
}

@container settings (max-width: 34rem) {
  .connection-rows__fields {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
