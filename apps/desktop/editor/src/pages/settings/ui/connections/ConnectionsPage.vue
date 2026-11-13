<script setup lang="ts">
/**
 * The page of connections: every connection by the order it was made, with its type and what it can
 * do, and the dialog that adds or edits one. A connection whose type no installed plugin provides is
 * listed as "Type unavailable". A row opens its connection for editing; the dialog holds Delete. The model of a connection is chosen where it is used, not here. After a deletion the focus goes to the row in its place, the row before it, or Add.
 */
import { nextTick, useTemplateRef } from 'vue'
import { SettingsPage } from '@numen/plugin-manager'
import { runInBackground } from '@numen/ui'
import type { ConnectionInfo } from '@/entities/connection'
import type { InstallationConnections } from '../../types'
import { useConnectionsPage } from '../../model/useConnectionsPage'
import { WORDS as words } from '../../words'
import { ConfirmDialog } from '../confirm-dialog'
import ConnectionDialog from './ConnectionDialog.vue'
import ConnectionList from './ConnectionList.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ connections: InstallationConnections }>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const page = useConnectionsPage(() => props.connections)
const { dialog, deletion } = page
const list = useTemplateRef<InstanceType<typeof ConnectionList>>('list')
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
      if (wasEditing) rowOpener?.focus()
      else list.value?.focusAdd()
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
      nextTick(() => list.value?.focusRow(at)),
      () => undefined,
    )
  })
}

function onCancelDelete(): void {
  deletion.cancel()
  deleteOpener?.focus()
}
</script>

<template>
  <SettingsPage :path="[words.settings, words.connections]" @back="onBack" @open="onBack">
    <div class="connections-page">
      <div v-if="page.connections.value.length === 0" class="connections-page__empty">
        <p class="connections-page__empty-title">{{ words.connectionsEmpty }}</p>
        <p class="connections-page__empty-detail">
          {{ page.hasType.value ? words.connectionsEmptyDetail : words.connectionsNoTypes }}
        </p>
      </div>
      <ConnectionList
        ref="list"
        :connections="page.connections.value"
        :types="page.types.value"
        :has-type="page.hasType.value"
        @edit="onEdit"
        @add="dialog.openAdd"
      />

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
  </SettingsPage>
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
