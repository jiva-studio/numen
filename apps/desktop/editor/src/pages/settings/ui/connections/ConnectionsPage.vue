<script setup lang="ts">
/**
 * The page of connections: every connection by the order it was made, with its type and what it can
 * do, and the dialog that adds or edits one. A connection whose type no installed plugin provides is
 * listed as "Type unavailable". The model of a connection is chosen where it is used, not here.
 */
import { computed, nextTick, onMounted, ref, useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import { Button, ConfirmDialog } from '@numen/ui'
import {
  asFailure,
  type ConnectionChange,
  type ConnectionDraft,
  type ConnectionInfo,
} from '@numen/core-client'
import { getUsedByNames, hasAvailableType, hasUnavailableConnection } from '@/entities/connection'
import type { SettingsTabState } from '../../types'
import { WORDS as words } from '../../words'
import ConnectionDialog from './ConnectionDialog.vue'
import ConnectionRow from './ConnectionRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: SettingsTabState }>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const connections = computed(() => installation.value.connections?.value ?? [])
const types = computed(() => installation.value.connectionTypes?.value ?? [])
const hasType = computed(() => hasAvailableType(types.value))
const hasUnavailable = computed(() => hasUnavailableConnection(connections.value))
const editing = ref<ConnectionInfo | null>(null)
const isAdding = ref(false)
const deleting = ref<ConnectionInfo | null>(null)
const deleteProblem = ref('')
const addButton = useTemplateRef<InstanceType<typeof Button>>('addButton')
let deleteOpener: HTMLElement | null = null

/* ---------------------------------- Hooks --------------------------------- */
onMounted(() => installation.value.loadConnections?.())

/* -------------------------------- Handlers -------------------------------- */
function onAdd(): void {
  isAdding.value = true
}

function onEdit(connection: ConnectionInfo): void {
  editing.value = connection
}

async function onCloseDialog(): Promise<void> {
  isAdding.value = false
  editing.value = null
  await nextTick()
  const element = addButton.value?.$el
  if (element instanceof HTMLElement) element.focus()
}

function onAskDelete(connection: ConnectionInfo, opener: HTMLElement): void {
  deleting.value = connection
  deleteProblem.value = ''
  deleteOpener = opener
}

async function onConfirmDelete(): Promise<void> {
  const target = deleting.value
  if (!target) return
  const answer = await installation.value.deleteConnection?.(target.id)
  if (answer && !answer.ok) {
    deleteProblem.value = answer.error
    return
  }
  onCancelDelete()
}

function onCancelDelete(): void {
  deleting.value = null
  deleteOpener?.focus()
}

/* -------------------------------- Helpers --------------------------------- */
function create(draft: ConnectionDraft) {
  const unavailable = asFailure(words.connectionsNoTypes)
  return installation.value.createConnection?.(draft) ?? Promise.resolve(unavailable)
}

function update(change: ConnectionChange) {
  const unavailable = asFailure(words.connectionsNoTypes)
  return installation.value.updateConnection?.(change) ?? Promise.resolve(unavailable)
}

function getDeleteText(connection: ConnectionInfo): string {
  const question = words.connectionDeleteAsking(connection.name, connection.secretKeysSet.length > 0)
  const usedBy = getUsedByNames(connection.uses)
  const text = usedBy === '' ? question : `${question} ${words.connectionUsedByDeleted(usedBy)}`
  return deleteProblem.value === ''
    ? text
    : `${text} ${words.connectionDeleteFailed} ${deleteProblem.value}`
}
</script>

<template>
  <div class="connections-page">
    <Button variant="ghost" size="small" class="connections-page__back" @click="emit('back')">
      &larr; {{ words.backToPlugins }}
    </Button>

    <div class="connections-page__head">
      <h2 class="connections-page__title">{{ words.connections }}</h2>
      <Button ref="addButton" variant="outline" :disabled="!hasType" @click="onAdd">
        <Plus :size="14" aria-hidden="true" />
        {{ words.connectionAdd }}
      </Button>
    </div>
    <p class="connections-page__intro">{{ words.connectionsIntro }}</p>

    <ul
      v-if="connections.length > 0"
      class="connections-page__list"
      :aria-label="words.connections"
    >
      <ConnectionRow
        v-for="connection in connections"
        :key="connection.id"
        :connection="connection"
        :type="types.find((one) => one.id === connection.typeId)"
        @edit="onEdit(connection)"
        @delete="(opener) => onAskDelete(connection, opener)"
      />
    </ul>
    <div v-else class="connections-page__empty">
      <p class="connections-page__empty-title">{{ words.connectionsEmpty }}</p>
      <p class="connections-page__empty-detail">
        {{ hasType ? words.connectionsEmptyDetail : words.connectionsNoTypes }}
      </p>
    </div>

    <p v-if="hasUnavailable" class="connections-page__legend">
      {{ words.connectionUnavailableDetail }}
    </p>

    <ConnectionDialog
      v-if="isAdding || editing"
      :types="types"
      v-bind="editing ? { connection: editing } : {}"
      :create="create"
      :update="update"
      @close="onCloseDialog"
    />

    <ConfirmDialog
      v-if="deleting"
      tone="danger"
      :text="getDeleteText(deleting)"
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
  justify-content: space-between;
}

.connections-page__title {
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 500;
}

.connections-page__intro,
.connections-page__legend {
  margin: 0.25rem 0 var(--settings-apart);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.connections-page__legend {
  margin-block: 0.75rem 0;
}

.connections-page__list {
  margin: 0;
  padding: 0;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
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
