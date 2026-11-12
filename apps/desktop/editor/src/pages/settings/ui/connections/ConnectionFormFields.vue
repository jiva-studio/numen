<script setup lang="ts">
/**
 * The part of the form of a connection that follows its type: the name, one input for each field
 * the type declares, and the notes. A type of an agent program has no
 * field and the form says so.
 */
import { computed } from 'vue'
import type {
  ConnectionFieldInfo,
  ConnectionTypeInfo,
  ConnectionValue,
} from '@/entities/connection'
import type { ConnectionFormView } from '../../model/connectionForm'
import { WORDS as words } from '../../words'
import ConnectionFieldInput from './ConnectionFieldInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  type: ConnectionTypeInfo
  view: ConnectionFormView
  values: Readonly<Record<string, ConnectionValue>>
  secrets: Readonly<Record<string, string>>
}>()

const name = defineModel<string>('name', { required: true })

const emit = defineEmits<{
  (e: 'update-value', key: string, value: ConnectionValue): void
  (e: 'update-secret', key: string, value: ConnectionValue): void
}>()

/* --------------------------------- State ---------------------------------- */
const hasNothingToEnter = computed(() => props.view.fields.length === 0)

/* -------------------------------- Handlers -------------------------------- */
function onUpdate(field: ConnectionFieldInfo, value: ConnectionValue): void {
  if (field.type === 'secret') emit('update-secret', field.key, value)
  else emit('update-value', field.key, value)
}

/* -------------------------------- Helpers --------------------------------- */
function getValue(field: ConnectionFieldInfo): ConnectionValue | undefined {
  return field.type === 'secret' ? props.secrets[field.key] : props.values[field.key]
}
</script>

<template>
  <div class="connection-form">
    <div class="connection-form__field">
      <label for="connection-name" class="connection-form__label">{{ words.connectionName }}</label>
      <input
        id="connection-name"
        v-model="name"
        class="connection-form__input"
        type="text"
        autocomplete="off"
        :placeholder="type.title"
      />
      <p class="connection-form__hint">{{ words.connectionNameHint }}</p>
    </div>

    <ConnectionFieldInput
      v-for="field in view.fields"
      :key="`${type.id}:${field.key}`"
      :field="field"
      :value="getValue(field)"
      :is-stored="field.type === 'secret' && view.storedKeys.includes(field.key)"
      @update="(value) => onUpdate(field, value)"
    />

    <p v-if="hasNothingToEnter" class="connection-form__hint">
      {{ words.connectionAgentNote(type.pluginName) }}
    </p>
    <p v-if="view.isKeyLost" role="status" class="connection-form__warning">
      {{ words.connectionAddressChanged }}
    </p>
  </div>
</template>

<style scoped>
.connection-form {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.connection-form__field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  align-items: flex-start;
}

.connection-form__label {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 500;
}

.connection-form__input {
  box-sizing: border-box;
  inline-size: 100%;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.connection-form__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.connection-form__hint,
.connection-form__warning {
  margin: 0;
  font-size: var(--numen-text-1);
}

.connection-form__hint {
  color: var(--numen-hushed);
}

.connection-form__warning {
  color: var(--numen-alarm);
}
</style>
