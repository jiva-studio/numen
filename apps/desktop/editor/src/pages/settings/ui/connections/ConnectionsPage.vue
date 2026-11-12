<script setup lang="ts">
/**
 * The page of connections: every connection by the order it was made, with its type and what it can
 * do, and the dialog that adds or edits one. A connection whose type no installed plugin provides is
 * listed as "Type unavailable". A row opens its connection for editing; the dialog holds Delete. The model of a connection is chosen where it is used, not here.
 */
import { nextTick, useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
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
let rowOpener: HTMLElement | null = null
let deleteOpener: HTMLElement | null = null

/* -------------------------------- Handlers -------------------------------- */
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
  deletion.confirm(() => deleteOpener?.focus())
}

function onCancelDelete(): void {
  deletion.cancel()
  deleteOpener?.focus()
}
</script>

<template>
  <div class="connections-page">
    <Button variant="ghost" size="small" class="connections-page__back" @click="emit('back')">
      &larr; {{ words.backToPlugins }}
    </Button>

    <div class="connections-page__head">
      <Button
        ref="addButton"
        variant="outline"
        :disabled="!page.hasType.value"
        @click="dialog.openAdd"
      >
        <Plus :size="14" aria-hidden="true" />
        {{ words.connectionAdd }}
      </Button>
    </div>

    <div v-if="page.connections.value.length > 0" class="connections-page__list" role="list">
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
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}

.connections-page__back {
  align-self: flex-start;
  margin-block-end: 0.75rem;
}

.connections-page__head {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  justify-content: flex-end;
}

.connections-page__head {
  margin-block-end: var(--settings-apart);
}


.connections-page__legend {
  margin: 0.25rem 0 var(--settings-apart);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.connections-page__legend {
  margin-block: 0.75rem 0;
}

.connections-page__list {
  display: flex;
  flex-direction: column;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
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
