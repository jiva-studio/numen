<script setup lang="ts">
/**
 * One field of the form of a connection, drawn by the type the manifest declares: a text, a number,
 * a switch, a choice of options, a file path, or a secret. A secret is a masked field that is empty
 * when the form opens, whether or not a key is stored; the stored key is never shown.
 */
import { computed, useId } from 'vue'
import { NumberField, Select, Switch } from '@numen/ui'
import type { SelectChoice } from '../../types'
import type { ConnectionFieldInfo, ConnectionValue } from '@/entities/connection'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  field: ConnectionFieldInfo
  value: ConnectionValue | undefined
  isStored?: boolean
}>()

const emit = defineEmits<{
  (e: 'update', value: ConnectionValue): void
}>()

/* --------------------------------- State ---------------------------------- */
const id = useId()
const choices = computed<readonly SelectChoice[]>(() =>
  props.field.options.map((one) => ({ id: one.value, text: one.title })),
)
const text = computed(() => (typeof props.value === 'string' ? props.value : ''))

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  if (event.target instanceof HTMLInputElement) emit('update', event.target.value)
}
</script>

<template>
  <div class="connection-field">
    <label :for="id" class="connection-field__label">{{ field.title }}</label>

    <Switch
      v-if="field.type === 'boolean'"
      :model-value="value === true"
      :aria-label="field.title"
      @update:model-value="(next) => emit('update', next)"
    />
    <NumberField
      v-else-if="field.type === 'number'"
      :id="id"
      :model-value="typeof value === 'number' ? value : null"
      :min="field.min ?? Number.MIN_SAFE_INTEGER"
      :max="field.max ?? Number.MAX_SAFE_INTEGER"
      :step="field.step ?? 1"
      @commit="(next) => next !== null && emit('update', next)"
    />
    <Select
      v-else-if="field.type === 'enum'"
      :model-value="text"
      :choices="choices"
      :name="field.title"
      :placeholder="words.connectionFieldSelect"
      @update:model-value="(next) => emit('update', next)"
    />
    <div v-else-if="field.type === 'secret'" class="connection-field__secret">
      <input
        :id="id"
        class="connection-field__input"
        type="password"
        autocomplete="off"
        :placeholder="isStored ? words.secretMasked : ''"
        :value="text"
        @input="onInput"
      />
    </div>
    <input
      v-else
      :id="id"
      class="connection-field__input"
      :class="{ 'connection-field__input--address': field.isAddress }"
      type="text"
      autocomplete="off"
      :value="text"
      @input="onInput"
    />

  </div>
</template>

<style scoped>
.connection-field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  align-items: flex-start;
}

.connection-field__label {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 500;
}

.connection-field__secret {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  inline-size: 100%;
}

.connection-field__input {
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

.connection-field__input--address {
  font-family: var(--numen-font-mono);
}

.connection-field__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

</style>
