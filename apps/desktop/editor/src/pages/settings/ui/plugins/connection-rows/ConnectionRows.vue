<script setup lang="ts">
/**
 * The rows of a setting of type connections: one line for each connection the plugin uses, in the
 * order of use. "Add connection" adds an empty row, whose connection list ends with "New connection…".
 * A setting of one row shows that row directly. A connection that cannot do the operation is dimmed
 * with the reason and listed after the others. A row is moved by dragging its grip, or by the
 * keyboard: Space picks it up, the arrows move it and Space drops it.
 *
 * With no connection at all, the rows give way to a note and "Open Connections", which asks the
 * settings tab to show the Connections page.
 *
 * The one row of a setting of one row that names no connection says "Choose a connection".
 * Only a row that names a connection and a value for each required row field of its type is written.
 * A row added and not yet complete stays on the page, and a row whose connection changed writes
 * nothing until its required row fields are chosen again.
 *
 * A change of the rows of a setting that indexes the vault indexes it again, which a service may
 * charge for. It is written after a question that names the notes and the connection, and a
 * row that is not complete changes nothing meanwhile.
 */
import { computed, nextTick, useTemplateRef, watch } from 'vue'
import { Button, runInBackground } from '@numen/ui'
import { PluginSettingsCard } from '@numen/plugin-manager'
import { getRowKey, readRows, type StoredRow } from '@/entities/connection'
import type { PluginSettingInfo } from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import type { InstallationConnections } from '../../../types'
import { ConfirmDialog } from '../../confirm-dialog'
import { WORDS as words } from '../../../words'
import ConnectionRowEditor from './ConnectionRowEditor.vue'
import ConnectionRowsHeader from './ConnectionRowsHeader.vue'
import { useConnectionEntries } from '../../../model/useConnectionEntries'
import { INDEXING_OPERATION, useIndexQuestion } from '../../../model/useIndexQuestion'
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

/* --------------------------------- State ---------------------------------- */
const list = useTemplateRef<HTMLElement>('list')
const { drafts, write, revert, add, remove, update, move } = useRowDrafts({
  getValue: () => props.value,
  getRowFields: (connectionId) => getRowFields(connectionId),
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

const operations = computed(() => props.setting.operations ?? [])
const limit = computed(() => props.setting.max ?? 1)
const { stored, choices, getReason, getRowFields } = useConnectionEntries(
  () => props.connections,
  () => operations.value,
)
const isSingle = computed(() => limit.value === 1)

/* A setting of one row shows its row at once, so that its connection is chosen in place. */
watch(
  [() => stored.value.length, () => drafts.value.length, limit],
  () => {
    if (isSingle.value && stored.value.length > 0 && drafts.value.length === 0) add()
  },
  { immediate: true },
)

/* -------------------------------- Handlers -------------------------------- */
function onAdd(): void {
  add()
  const added = drafts.value[drafts.value.length - 1]
  if (!added) return
  runInBackground(
    nextTick(() =>
      list.value
        ?.querySelector<HTMLElement>(
          `[data-row-key="${added.key}"] [aria-label="${words.connectionRowsConnection}"]`,
        )
        ?.focus(),
    ),
    () => undefined,
  )
}

function onRowsChange(rows: readonly StoredRow[]): void {
  if (!operations.value.includes(INDEXING_OPERATION)) {
    emit('change', rows)
    return
  }
  if (drafts.value.length !== rows.length) return
  if (isIndexChange(rows)) ask(rows)
  else emit('change', rows)
}

/* -------------------------------- Helpers --------------------------------- */
function getRowReason(connection: string): string {
  if (connection === '' && isSingle.value) return words.connectionRowsChoose
  return getReason(connection)
}

function isIndexChange(rows: readonly StoredRow[]): boolean {
  const written = readRows(props.value).map(getRowKey)
  return rows.length > 0 && rows.map(getRowKey).join('\n') !== written.join('\n')
}
</script>

<template>
  <section class="connection-rows">
    <PluginSettingsCard>
      <ConnectionRowsHeader
        :at="setting.key"
        :name="setting.title"
        :can-add="stored.length > 0 && !isSingle"
        :is-add-disabled="drafts.length >= limit"
        @add="onAdd"
      />

      <div v-if="stored.length === 0" class="connection-rows__empty">
        <p class="connection-rows__note">{{ words.connectionRowsNone }}</p>
        <Button variant="outline" @click="emit('open-connections')">
          {{ words.connectionRowsOpen }}
        </Button>
      </div>

      <ul
        v-else-if="drafts.length > 0"
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
          :operations="operations"
          :choices="choices"
          :reason="getRowReason(row.connection)"
          :row-fields="getRowFields(row.connection)"
          :is-single="isSingle"
          :types="props.connections.types.value"
          :create="props.connections.edits.create"
          :update="props.connections.edits.update"
          :is-held="heldKey === row.key"
          :is-dragging="
            drag.dragging.value?.hasMoved === true && drag.dragging.value.item === index
          "
          :is-drop-target="drag.dragging.value?.hasMoved === true && drag.at.value === index"
          :list-row-options="props.connections.listRowOptions"
          @update="(patch) => update(row.key, patch)"
          @remove="remove(row.key)"
          @grip-press="(event) => onGripPress(index, event)"
          @grip-key="(event) => onGripKey(row.key, event)"
        />
      </ul>
    </PluginSettingsCard>

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
  </section>
</template>

<style scoped>
.connection-rows {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  inline-size: 100%;
  margin-block-start: 1.25rem;
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
  padding-block: 0.625rem;
}

.connection-rows__note {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
