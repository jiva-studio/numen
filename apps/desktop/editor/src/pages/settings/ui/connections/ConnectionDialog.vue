<script setup lang="ts">
/**
 * The dialog that adds a connection or edits one. The form is built from the fields the chosen type
 * declares: a type that fixes its address shows no address, a type of an agent program shows no
 * field at all. The model is not part of a connection. A key is typed here once and is never shown
 * again; changing the address removes the stored key. The type is chosen from a list that filters
 * as one types. A saved connection is announced with `saved` before the dialog closes.
 */
import { Trash2 } from '@lucide/vue'
import { computed, onMounted, useId, useTemplateRef } from 'vue'
import { Button, Select } from '@numen/ui'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionInfo,
  ConnectionTypeInfo,
} from '@/entities/connection'
import type { Result } from '@/shared/lib/result'
import type { SelectChoice } from '../../types'
import { useConnectionForm } from '../../model/connectionForm'
import { WORDS as words } from '../../words'
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
const titleId = useId()
const dialog = useTemplateRef<HTMLElement>('dialog')
const form = useConnectionForm({
  types: () => props.types,
  connection: () => props.connection,
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

/* ---------------------------------- Hooks --------------------------------- */
onMounted(() => dialog.value?.querySelector<HTMLElement>('input, [role="combobox"]')?.focus())

/* -------------------------------- Handlers -------------------------------- */
function onKeepFocus(event: KeyboardEvent): void {
  const items = Array.from(
    dialog.value?.querySelectorAll<HTMLElement>(
      'input, button, [role="combobox"], [role="switch"], a[href]',
    ) ?? [],
  ).filter((one) => !one.hasAttribute('disabled'))
  const first = items[0]
  const last = items[items.length - 1]
  if (!first || !last) return
  if (event.shiftKey && event.target === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && event.target === last) {
    event.preventDefault()
    first.focus()
  }
}
</script>

<template>
  <div
    class="connection-dialog__backdrop"
    @click.self="emit('close')"
    @keydown.esc="emit('close')"
    @keydown.tab="onKeepFocus"
  >
    <form
      ref="dialog"
      class="connection-dialog"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="titleId"
      @submit.prevent="form.actions.submit"
    >
      <h2 :id="titleId" class="connection-dialog__title">
        {{ form.isEditing ? words.connectionEdit : words.connectionAdd }}
      </h2>

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

      <div class="connection-dialog__actions">
        <Button
          v-if="connection"
          type="button"
          variant="outline"
          class="connection-dialog__delete"
          :aria-label="words.connectionMenuDelete"
          :title="words.connectionMenuDelete"
          @click="emit('delete', connection)"
        >
          <Trash2 :size="14" aria-hidden="true" />
          {{ words.connectionMenuDelete }}
        </Button>
        <Button type="button" variant="outline" @click="emit('close')">
          {{ words.connectionCancel }}
        </Button>
        <Button type="submit" variant="solid" :disabled="!type || form.state.isSaving">
          {{ words.connectionSave }}
        </Button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.connection-dialog__backdrop {
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--numen-gutter);
  background: color-mix(in oklab, var(--numen-surface), transparent 30%);
}

.connection-dialog {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  box-sizing: border-box;
  inline-size: 100%;
  max-inline-size: 28rem;
  max-block-size: 100%;
  overflow-y: auto;
  padding: 1rem 1.25rem;
  background: var(--numen-raised);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
}

.connection-dialog__title {
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 500;
}

.connection-dialog__field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  align-items: flex-start;
}

.connection-dialog__label {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 500;
}

.connection-dialog__hint,
.connection-dialog__problem {
  margin: 0;
  font-size: var(--numen-text-1);
}

.connection-dialog__hint {
  color: var(--numen-hushed);
}

.connection-dialog__problem {
  color: var(--numen-alarm);
}

.connection-dialog__delete {
  margin-inline-end: auto;
}

.connection-dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.375rem;
}
</style>
