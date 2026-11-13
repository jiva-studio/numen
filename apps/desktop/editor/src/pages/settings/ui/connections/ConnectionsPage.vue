<script setup lang="ts">
/**
 * The page of connections: every connection by the order it was made, with its type and what it can
 * do, and the dialog that adds or edits one. A connection whose type no installed plugin provides is
 * listed as "Type unavailable". The last row of the list adds a connection, as the add button row of a GNOME
 * boxed list. A row opens its connection for editing; the dialog holds Delete. The model of a connection is chosen where it is used, not here. After a deletion the focus goes to the row in its place, the row before it, or Add.
 */
import { nextTick, useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import { SettingsPage } from '@numen/plugin-manager'
import { runInBackground } from '@numen/ui'
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
const addButton = useTemplateRef<HTMLButtonElement>('addButton')
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
      const element = wasEditing ? rowOpener : addButton.value
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
  const element = rows[at] ?? rows[at - 1] ?? addButton.value
  if (element instanceof HTMLElement) element.focus()
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
      <div class="settings-card connections-page__list">
        <div ref="list" role="list" class="connections-page__rows">
          <ConnectionRow
            v-for="connection in page.connections.value"
            :key="connection.id"
            :connection="connection"
            :type="page.types.value.find((one) => one.id === connection.typeId)"
            @edit="(opener) => onEdit(connection, opener)"
          />
        </div>
        <button
          ref="addButton"
          type="button"
          class="connections-page__add"
          :disabled="!page.hasType.value"
          @click="dialog.openAdd"
        >
          <Plus
            class="connections-page__add-glyph"
            :size="20"
            :stroke-width="1.75"
            aria-hidden="true"
          />
          {{ words.connectionAddRow }}
        </button>
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

.connections-page__list {
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
}

.connections-page__rows {
  display: flex;
  flex-direction: column;
}

.connections-page__add {
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

.connections-page__rows:not(:empty) + .connections-page__add {
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
}

.connections-page__add:hover:not(:disabled) {
  background-color: var(--settings-hover-bg);
}

.connections-page__add:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: calc(-1 * var(--numen-stroke));
}

.connections-page__add:disabled {
  color: var(--numen-hushed);
  cursor: default;
}

.connections-page__add-glyph {
  flex: none;
  color: var(--numen-hushed);
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
