<script setup lang="ts">
/**
 * The dialog that adds a connection or edits one. The form is built from the fields the chosen type
 * declares: a type that fixes its address shows no address, a type of an agent program shows no
 * field at all. The model is not part of a connection. A key is typed here once and is never shown
 * again; changing the address removes the stored key. The type is chosen from a list that filters
 * as one types. A saved connection is announced with `saved` before the dialog closes.
 */
import { computed } from 'vue'
import { ModalDialog, Select } from '@numen/ui'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionInfo,
  ConnectionTypeInfo,
} from '@/entities/connection'
import type { Result } from '@numen/result'
import type { SelectChoice } from '../../types'
import { useConnectionForm } from '../../model/connectionForm'
import { WORDS as words } from '../../words'
import ConnectionDialogActions from './ConnectionDialogActions.vue'
import ConnectionFormFields from './ConnectionFormFields.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  types: readonly ConnectionTypeInfo[]
  connection?: ConnectionInfo
  create: (draft: ConnectionDraft) => Promise<Result<ConnectionInfo, string>>
  update: (change: ConnectionChange) => Promise<Result<ConnectionInfo, string>>
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', connection: ConnectionInfo): void
  (e: 'delete', connection: ConnectionInfo): void
}>()

/* --------------------------------- State ---------------------------------- */
const form = useConnectionForm({
  listTypes: () => props.types,
  getConnection: () => props.connection,
  create: props.create,
  update: props.update,
  onSaved: (saved) => {
    emit('saved', saved)
    emit('close')
  },
})
const typeChoices = computed<readonly SelectChoice[]>(() =>
  form.availableTypes.value.map((one) => ({ id: one.id, text: one.title })),
)
const type = computed(() => form.view.value.type)

/* -------------------------------- Handlers -------------------------------- */
function onDelete(): void {
  if (props.connection) emit('delete', props.connection)
}
</script>

<template>
  <ModalDialog
    is-form
    :label="form.isEditing ? words.connectionEdit : words.connectionAdd"
    @submit="form.actions.submit"
    @cancel="emit('close')"
  >
    <div class="connection-dialog__field">
      <span class="connection-dialog__label">{{ words.connectionType }}</span>
      <Select
        :model-value="form.state.typeId"
        :choices="typeChoices"
        :name="words.connectionType"
        :search-name="words.searchConnectionTypes"
        :is-disabled="form.isEditing"
        is-searchable
        @update:model-value="form.actions.chooseType"
      />
    </div>

    <ConnectionFormFields
      v-if="type"
      v-model:name="form.state.name"
      :type="type"
      :view="form.view.value"
      :values="form.state.values"
      :secrets="form.state.secrets"
      @update-value="form.actions.updateValue"
      @update-secret="form.actions.updateSecret"
    />
    <p v-else class="connection-dialog__hint">{{ words.connectionsNoTypes }}</p>

    <p v-if="form.state.problem !== ''" role="alert" class="connection-dialog__problem">
      {{ form.state.problem }}
    </p>

    <template #actions>
      <ConnectionDialogActions
        :can-delete="connection !== undefined"
        :can-save="type !== undefined && !form.state.isSaving"
        @cancel="emit('close')"
        @delete="onDelete"
      />
    </template>
  </ModalDialog>
</template>

<style scoped>
.connection-dialog__field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  align-items: flex-start;
}

.connection-dialog__label {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  font-weight: 500;
}

.connection-dialog__hint,
.connection-dialog__problem {
  margin: 0;
  font-size: var(--numen-text-2);
}

.connection-dialog__hint {
  color: var(--numen-hushed);
}

.connection-dialog__problem {
  color: var(--numen-alarm);
}
</style>
