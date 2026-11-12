<script setup lang="ts">
/**
 * The rows of a setting of type connections: one line for each connection the plugin uses, in the
 * order of use. The person uses a connection that exists: "Use a connection" opens the list of
 * them, which ends with "New connection…" (it opens the dialog that adds one and uses the result).
 * A setting of one row shows that row directly. A connection that cannot do the operation is dimmed
 * with the reason and listed after the others. A row is moved by dragging its grip, or by the
 * keyboard: Space picks it up, the arrows move it and Space drops it.
 *
 * With no connection at all, the rows give way to a note and "Open Connections", which asks the
 * settings tab to show the Connections page.
 *
 * Only a row that names a connection and a model is written. A row added and not yet complete
 * stays on the page.
 *
 * A change of the connection or the model of embeddings indexes the vault again, which a service
 * may charge for. It is written after a question that names the notes and the connection, and a
 * row that is not complete changes nothing meanwhile.
 */
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue'
import { Button, Select, runInBackground } from '@numen/ui'
import { readRows, type ConnectionInfo, type StoredRow } from '@/entities/connection'
import type { PluginSettingInfo } from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import type { InstallationConnections } from '../../../types'
import type { SelectChoice } from '../../../types'
import { ConfirmDialog } from '../../confirm-dialog'
import ConnectionDialog from '../../connections/ConnectionDialog.vue'
import { WORDS as words } from '../../../words'
import ConnectionRowEditor from './ConnectionRowEditor.vue'
import { useConnectionEntries } from '../../../model/useConnectionEntries'
import { openSelect } from '../../../model/openSelect'
import { useIndexQuestion } from '../../../model/useIndexQuestion'
import { useRowDrafts } from '../../../model/useRowDrafts'
import { useRowOrder } from '../../../model/useRowOrder'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  setting: PluginSettingInfo
  value: JsonValue | undefined
  connections: InstallationConnections
}>()

const emit = defineEmits<{
  (e: 'change', rows: readonly StoredRow[]): void
  (e: 'open-connections'): void
}>()

/** The identifier of the entry that adds a connection; no stored connection has it. */
const NEW_CONNECTION = 'new-connection'

/* --------------------------------- State ---------------------------------- */
const list = useTemplateRef<HTMLElement>('list')
const { drafts, write, revert, add, remove, update, move } = useRowDrafts({
  getValue: () => props.value,
  onChange: (rows) => onRowsChange(rows),
})
const { question, ask, confirm, cancel } = useIndexQuestion({
  getConnections: () => props.connections,
  getText: (rows, count) =>
    words.connectionRowsIndexQuestion(
      count,
      stored.value.find((one) => one.id === rows[0]?.connection)?.name ?? '',
    ),
  onConfirm: (rows) => emit('change', rows),
  onCancel: revert,
})
const { heldKey, announcement, drag, onGripPress, onGripKey } = useRowOrder({
  drafts,
  getList: () => list.value,
  write,
  move,
})

const operation = computed(() => props.setting.operation ?? '')
const limit = computed(() => props.setting.max ?? 1)
const { stored, choices, getReason } = useConnectionEntries(
  () => props.connections,
  () => operation.value,
)
const isSingle = computed(() => limit.value === 1)
const isAdding = ref(false)
const pickerChoices = computed<readonly SelectChoice[]>(() => [
  ...choices.value,
  { id: NEW_CONNECTION, text: words.connectionNew },
])

/* A setting of one row shows its row at once, so that its connection is chosen in place. */
watch(
  [() => stored.value.length, () => drafts.value.length, limit],
  () => {
    if (isSingle.value && stored.value.length > 0 && drafts.value.length === 0) add()
  },
  { immediate: true },
)

/* -------------------------------- Handlers -------------------------------- */
function onUse(id: string): void {
  if (id === NEW_CONNECTION) {
    isAdding.value = true
    return
  }
  addRow(id)
}

function onConnectionSaved(saved: ConnectionInfo): void {
  addRow(saved.id)
}

function addRow(connection: string): void {
  add()
  const added = drafts.value[drafts.value.length - 1]
  if (!added) return
  update(added.key, { connection })
  runInBackground(
    nextTick(() =>
      openSelect(
        list.value?.querySelector<HTMLElement>(
          `[data-row-key="${added.key}"] [aria-label="${words.connectionRowsModel}"]`,
        ),
      ),
    ),
    () => undefined,
  )
}

function onRowsChange(rows: readonly StoredRow[]): void {
  if (operation.value !== 'embed') {
    emit('change', rows)
    return
  }
  if (drafts.value.length !== rows.length) return
  if (isIndexChange(rows)) ask(rows)
  else emit('change', rows)
}

/* -------------------------------- Helpers --------------------------------- */
function isIndexChange(rows: readonly StoredRow[]): boolean {
  const getKey = (one: StoredRow): string => `${one.connection}\0${one.model}`
  const written = readRows(props.value).map(getKey)
  return rows.length > 0 && rows.map(getKey).join('\n') !== written.join('\n')
}
</script>

<template>
  <div class="connection-rows">
    <div v-if="stored.length === 0" class="connection-rows__empty">
      <p class="connection-rows__note">{{ words.connectionRowsNone }}</p>
      <Button variant="outline" @click="emit('open-connections')">
        {{ words.connectionRowsOpen }}
      </Button>
    </div>

    <ul
      v-if="stored.length > 0 && drafts.length > 0"
      ref="list"
      class="connection-rows__list"
      :aria-label="setting.title"
    >
      <ConnectionRowEditor
        v-for="(row, index) in drafts"
        :key="row.key"
        :row="row"
        :position="index + 1"
        :count="drafts.length"
        :operation="operation"
        :choices="choices"
        :reason="getReason(row.connection)"
        :is-single="isSingle"
        :types="props.connections.types.value"
        :create="props.connections.create"
        :update="props.connections.update"
        :is-held="heldKey === row.key"
        :is-dragging="drag.dragging.value?.hasMoved === true && drag.dragging.value.item === index"
        :is-drop-target="drag.dragging.value?.hasMoved === true && drag.at.value === index"
        :list-models="props.connections.models.listModels"
        :list-reasoning-levels="props.connections.models.listReasoningLevels"
        @update="(patch) => update(row.key, patch)"
        @remove="remove(row.key)"
        @grip-press="(event) => onGripPress(index, event)"
        @grip-key="(event) => onGripKey(row.key, event)"
      />
    </ul>

    <div v-if="stored.length > 0 && !isSingle && drafts.length < limit" class="connection-rows__foot">
      <Select
        :model-value="''"
        :choices="pickerChoices"
        :name="words.connectionRowsUse"
        :placeholder="words.connectionRowsUse"
        :aria-label="words.connectionRowsUse"
        :search-name="words.searchConnections"
        is-searchable
        class="connection-rows__picker"
        @update:model-value="onUse"
      />
    </div>

    <ConnectionDialog
      v-if="isAdding"
      :types="props.connections.types.value"
      :create="props.connections.create"
      :update="props.connections.update"
      @saved="onConnectionSaved"
      @close="isAdding = false"
    />

    <p v-if="stored.length > 0 && drafts.length > 0" class="connection-rows__note">
      {{ words.connectionRowsPrivacy(limit) }}
    </p>
    <p class="sr-only" role="status" aria-live="assertive">{{ announcement }}</p>

    <ConfirmDialog
      v-if="question"
      :text="question.text"
      :label="words.connectionRowsIndexLabel"
      :confirm-text="words.connectionRowsIndexConfirm"
      :cancel-text="words.connectionCancel"
      @confirm="confirm"
      @cancel="cancel"
    />
  </div>
</template>

<style scoped>
.connection-rows {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  inline-size: 100%;
}

.connection-rows__list {
  margin: 0;
  padding: 0;
}

.connection-rows__empty {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 0.75rem;
  align-items: center;
}

.connection-rows__picker {
  inline-size: 14rem;
  max-inline-size: 100%;
}

.connection-rows__foot {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
}

.connection-rows__note {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
