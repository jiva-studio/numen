<script setup lang="ts">
/**
 * One row of a setting of rows: the grip that moves it, the connection, the model and the reasoning
 * level, and the button that removes it. The models and the levels are asked from the provider of the
 * connection; the model field also takes a typed name. Choosing a connection fills the model with the
 * first model of its list. The list of connections ends with "New connection…", which opens the
 * dialog that adds one; the saved connection is chosen in the row and the list of its models opens,
 * so the person chooses the model at once. Closing the dialog leaves the row as it was.
 */
import { computed, nextTick, ref, useTemplateRef } from 'vue'
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
const { modelChoices, levelChoices, problem } = useRowChoices({
  getRow: () => props.row,
  getOperation: () => props.operation,
  getListModels: () => props.listModels,
  getListReasoningLevels: () => props.listReasoningLevels,
  update: (patch) => emit('update', patch),
})

const isAdding = ref(false)
const modelSelect = useTemplateRef<InstanceType<typeof Select>>('modelSelect')
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
  emit('update', { connection, model: '', reasoning: '' })
}

function onConnectionSaved(saved: ConnectionInfo): void {
  emit('update', { connection: saved.id, model: '', reasoning: '' })
  runInBackground(
    nextTick(() => openSelect(modelSelect.value?.element)),
    () => undefined,
  )
}

function onModelChoose(model: string): void {
  if (model === props.row.model) return
  emit('update', { model, reasoning: '' })
}

function onReasoningChoose(reasoning: string): void {
  emit('update', { reasoning })
}
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

    <div class="connection-rows__fields">
      <Select
        :model-value="row.connection"
        :choices="connectionChoices"
        :name="words.connectionRowsConnection"
        :placeholder="words.connectionRowsChoose"
        :aria-label="words.connectionRowsConnection"
        :search-name="words.searchConnections"
        is-searchable
        class="connection-rows__select"
        @update:model-value="onConnectionChoose"
      />
      <Select
        ref="modelSelect"
        :model-value="row.model"
        :choices="modelChoices"
        :name="words.connectionRowsModel"
        :placeholder="words.connectionRowsModelPlaceholder"
        :aria-label="words.connectionRowsModel"
        :search-name="words.searchModels"
        :is-disabled="row.connection === ''"
        is-searchable
        can-enter-custom
        class="connection-rows__select"
        @update:model-value="onModelChoose"
      />
      <Select
        v-if="levelChoices.length > 1 && row.connection !== '' && row.model !== ''"
        :model-value="row.reasoning"
        :choices="levelChoices"
        :name="words.connectionRowsReasoning"
        :aria-label="words.connectionRowsReasoning"
        class="connection-rows__select connection-rows__select--narrow"
        @update:model-value="onReasoningChoose"
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

    <p v-if="reason !== ''" class="connection-rows__problem">{{ reason }}</p>
    <p v-if="problem !== ''" class="connection-rows__problem">
      {{ words.connectionRowsModelsFailed }} {{ problem }}
    </p>
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
  display: flex;
  flex: 1 1 0;
  flex-wrap: wrap;
  gap: 0.375rem;
  min-inline-size: 0;
}

.connection-rows__select {
  flex: 1 1 7rem;
  min-inline-size: 0;
}

.connection-rows__select--narrow {
  flex: 0 1 6rem;
}

.connection-rows__problem {
  flex: 1 0 100%;
  order: 9;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
