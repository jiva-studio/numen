<script setup lang="ts">
/**
 * The page of connections: every connection by the order it was made, with its type and what it can
 * do, and the dialog that adds or edits one. A connection whose type no installed plugin provides is
 * listed as "Type unavailable". A row opens its connection for editing; the dialog holds Delete. The model of a connection is chosen where it is used, not here. After a deletion the focus goes to the row in its place, the row before it, or Add.
 */
import { nextTick, useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import { SettingsToolbar } from '@numen/plugin-manager'
import { Button, runInBackground } from '@numen/ui'
import type { ConnectionInfo } from '@/entities/connection'
import type { InstallationConnections } from '../../types'
import { useConnectionsPage } from '../../model/useConnectionsPage'
import { WORDS as words } from '../../words'
import { ConfirmDialog } from '../confirm-dialog'
import ConnectionDialog from './ConnectionDialog.vue'
import ConnectionRow from './ConnectionRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ connections: InstallationConnections }>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const page = useConnectionsPage(() => props.connections)
const { dialog, deletion } = page
const addButton = useTemplateRef<InstanceType<typeof Button>>('addButton')
const list = useTemplateRef<HTMLElement>('list')
let rowOpener: HTMLElement | null = null
let deleteOpener: HTMLElement | null = null

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onCloseDialog(): void {
  const wasEditing = dialog.editing.value !== null
  dialog.close()
  runInBackground(
    nextTick(() => {
      const element = wasEditing ? rowOpener : addButton.value?.$el
      if (element instanceof HTMLElement) element.focus()
    }),
    () => undefined,
  )
}

function onEdit(connection: ConnectionInfo, opener: HTMLElement): void {
  rowOpener = opener
  dialog.openEdit(connection)
}

function onAskDelete(connection: ConnectionInfo): void {
  deletion.ask(connection)
  deleteOpener = rowOpener
  dialog.close()
}

function onConfirmDelete(): void {
  const at = page.connections.value.findIndex((one) => one.id === deletion.target.value?.id)
  deletion.confirm(() => {
    runInBackground(
      nextTick(() => focusAfterDelete(at)),
      () => undefined,
    )
  })
}

function onCancelDelete(): void {
  deletion.cancel()
  deleteOpener?.focus()
}

/* -------------------------------- Helpers --------------------------------- */
function focusAfterDelete(at: number): void {
  const rows = Array.from(list.value?.children ?? [])
  const element = rows[at] ?? rows[at - 1] ?? addButton.value?.$el
  if (element instanceof HTMLElement) element.focus()
}
</script>

<template>
  <div class="connections-page">
    <SettingsToolbar :path="[words.settings, words.connections]" @back="onBack" @open="onBack">
      <Button
        ref="addButton"
        variant="ghost"
        size="small"
        :disabled="!page.hasType.value"
        @click="dialog.openAdd"
      >
        <Plus :size="14" aria-hidden="true" />
        {{ words.connectionAdd }}
      </Button>
    </SettingsToolbar>

    <div
      v-if="page.connections.value.length > 0"
      ref="list"
      class="settings-card connections-page__list"
      role="list"
    >
      <ConnectionRow
        v-for="connection in page.connections.value"
        :key="connection.id"
        :connection="connection"
        :type="page.types.value.find((one) => one.id === connection.typeId)"
        @edit="(opener) => onEdit(connection, opener)"
      />
    </div>
    <div v-else class="connections-page__empty">
      <p class="connections-page__empty-title">{{ words.connectionsEmpty }}</p>
      <p class="connections-page__empty-detail">
        {{ page.hasType.value ? words.connectionsEmptyDetail : words.connectionsNoTypes }}
      </p>
    </div>

    <p v-if="page.hasUnavailable.value" class="connections-page__legend">
      {{ words.connectionUnavailableDetail }}
    </p>

    <ConnectionDialog
      v-if="dialog.isOpen.value"
      :types="page.types.value"
      v-bind="dialog.editing.value ? { connection: dialog.editing.value } : {}"
      :create="page.create"
      :update="page.update"
      @delete="onAskDelete"
      @close="onCloseDialog"
    />

    <ConfirmDialog
      v-if="deletion.target.value"
      tone="danger"
      :text="deletion.text.value"
      :label="words.connectionDelete"
      :confirm-text="words.connectionDeleteConfirm"
      :cancel-text="words.connectionDeleteCancel"
      @confirm="onConfirmDelete"
      @cancel="onCancelDelete"
    />
  </div>
</template>

<style scoped>
.connections-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}

.connections-page__legend {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.connections-page__list {
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
}

.connections-page__empty {
  padding: 1.5rem 1rem;
  border: var(--numen-stroke) dashed var(--numen-rule);
  border-radius: var(--numen-radius);
  text-align: center;
}

.connections-page__empty-title {
  margin: 0 0 0.25rem;
  font-weight: 500;
}

.connections-page__empty-detail {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
