<script setup lang="ts">
/**
 * The rows of a setting of type connections: one row for each connection the plugin tries, in the
 * order of use, with "Add connection" until the setting holds `max` rows. Only existing connections
 * can be chosen; one that cannot do the operation is dimmed with the reason and listed after the
 * others. A row is moved by dragging its grip, or by the keyboard: Space picks it up, the arrows
 * move it and Space drops it.
 *
 * With no row, "Open Connections" asks the settings tab to show the Connections page.
 *
 * Only a row that names a connection and a model is written. A row added and not yet complete
 * stays on the page.
 *
 * A change of the connection or the model of embeddings indexes the vault again, which a service
 * may charge for. It is written after a question that names the notes and the connection, and a
 * row that is not complete changes nothing meanwhile.
 */
import { computed, useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import { Button } from '@numen/ui'
import { readRows, type StoredRow } from '@/entities/connection'
import type { PluginSettingInfo } from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import type { InstallationConnections } from '../../../types'
import { ConfirmDialog } from '../../confirm-dialog'
import { WORDS as words } from '../../../words'
import ConnectionRowEditor from './ConnectionRowEditor.vue'
import { useConnectionEntries } from '../../../model/useConnectionEntries'
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
const canAdd = computed(() => drafts.value.length < limit.value && stored.value.length > 0)

/* -------------------------------- Handlers -------------------------------- */
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
    <ul
      v-if="drafts.length > 0"
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

    <div class="connection-rows__foot">
      <Button v-if="drafts.length < limit" variant="outline" :disabled="!canAdd" @click="add">
        <Plus :size="14" aria-hidden="true" />
        {{ words.connectionAdd }}
      </Button>
      <Button v-if="drafts.length === 0" variant="ghost" @click="emit('open-connections')">
        {{ words.connectionRowsOpen }}
      </Button>
      <p v-if="stored.length === 0" class="connection-rows__note">
        {{ words.connectionRowsNone }}
      </p>
    </div>

    <p v-if="drafts.length > 0" class="connection-rows__note">
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
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  overflow: hidden;
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
