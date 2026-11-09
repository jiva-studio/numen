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
 */
import { computed, onMounted, useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import { Button, type SelectChoice } from '@numen/ui'
import {
  getConnectionUsability,
  sortByUsability,
  type RowDraft,
  type StoredRow,
} from '@/entities/connection'
import type { PluginSettingInfo } from '@/entities/settings'
import type { Installation } from '../../../types'
import { WORDS as words } from '../../../words'
import ConnectionRowEditor from './ConnectionRowEditor.vue'
import { useRowDrafts } from './useRowDrafts'
import { useRowOrder } from './useRowOrder'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  setting: PluginSettingInfo
  value: unknown
  installation: Installation
}>()

const emit = defineEmits<{
  (e: 'change', rows: readonly StoredRow[]): void
  (e: 'open-connections'): void
}>()

/* --------------------------------- State ---------------------------------- */
const list = useTemplateRef<HTMLElement>('list')
const { drafts, write, add, remove, update, move } = useRowDrafts({
  getValue: () => props.value,
  onChange: (rows) => emit('change', rows),
})
const { heldKey, announcement, drag, onGripPress, onGripKey } = useRowOrder({
  drafts,
  getList: () => list.value,
  write,
  move,
})

const operation = computed(() => props.setting.operation ?? '')
const limit = computed(() => props.setting.max ?? 1)
const connections = computed(() => props.installation.connections?.value ?? [])
const types = computed(() => props.installation.connectionTypes?.value ?? [])
const usabilityWords = {
  unavailable: words.connectionUnavailable,
  incomplete: words.connectionIncomplete,
  cannot: words.connectionCannot,
}

const entries = computed(() =>
  sortByUsability(
    connections.value.map((connection) => {
      const type = types.value.find((one) => one.id === connection.typeId)
      return {
        connection,
        type,
        ...getConnectionUsability(connection, type, operation.value, usabilityWords),
      }
    }),
  ),
)

const choices = computed<readonly SelectChoice[]>(() =>
  entries.value.map((one) => ({
    id: one.connection.id,
    text: one.connection.name,
    detail: one.isUsable ? (one.type?.title ?? '') : one.reason,
    ...(one.isUsable ? {} : { isDisabled: true }),
  })),
)

const canAdd = computed(() => drafts.value.length < limit.value && connections.value.length > 0)

/* ---------------------------------- Hooks --------------------------------- */
onMounted(() => props.installation.loadConnections?.())

/* -------------------------------- Helpers --------------------------------- */
function getReason(row: RowDraft): string {
  return entries.value.find((one) => one.connection.id === row.connection)?.reason ?? ''
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
        :reason="getReason(row)"
        :is-held="heldKey === row.key"
        :is-dragging="drag.dragging.value?.hasMoved === true && drag.dragging.value.item === index"
        :is-drop-target="drag.dragging.value?.hasMoved === true && drag.at.value === index"
        v-bind="{
          ...(installation.listModels ? { listModels: installation.listModels } : {}),
          ...(installation.listReasoningLevels
            ? { listReasoningLevels: installation.listReasoningLevels }
            : {}),
        }"
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
      <p v-if="connections.length === 0" class="connection-rows__note">
        {{ words.connectionRowsNone }}
      </p>
    </div>

    <p v-if="drafts.length > 0" class="connection-rows__note">{{ words.connectionRowsPrivacy }}</p>
    <p class="sr-only" role="status" aria-live="assertive">{{ announcement }}</p>
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
