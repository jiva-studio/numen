<script setup lang="ts">
/**
 * The plugin declarations that may read a secret: a button that reads the number of plugins and
 * opens a checklist with a check box for each declaration, labelled with the plugin title.
 */
import { computed, ref } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { Checkbox } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import { isAllowed, type Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  /** Absent while the secret does not exist yet: the list is then disabled. */
  secret?: Secret | undefined
  declarations: readonly Declaration[]
}>()

const emit = defineEmits<{
  (e: 'toggle', declaration: Declaration, isAllowed: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const count = computed(() => {
  const secret = props.secret
  return secret ? props.declarations.filter((one) => isAllowed(secret, one)).length : 0
})
</script>

<template>
  <div class="secret-permissions" @keydown.esc="isOpen = false">
    <button
      type="button"
      class="secret-permissions__button"
      :disabled="!secret || declarations.length === 0"
      :title="secret ? undefined : words.secretSaveFirst"
      :aria-expanded="isOpen"
      @click="isOpen = !isOpen"
    >
      {{ words.secretReaders(count) }}
      <ChevronDown :size="14" aria-hidden="true" />
    </button>
    <div v-if="isOpen" class="secret-permissions__backdrop" @click="isOpen = false" />
    <div
      v-if="isOpen"
      class="secret-permissions__list"
      role="group"
      :aria-label="words.secretAllowedTo"
    >
      <Checkbox
        v-for="one in declarations"
        :key="`${one.pluginId}/${one.name}/${one.group}`"
        class="secret-permissions__declaration"
        :title="one.description"
        :model-value="secret !== undefined && isAllowed(secret, one)"
        @update:model-value="(value) => emit('toggle', one, value)"
      >
        {{ one.pluginName }}
      </Checkbox>
    </div>
  </div>
</template>

<style scoped>
.secret-permissions {
  position: relative;
}

.secret-permissions__button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.25rem;
  box-sizing: border-box;
  inline-size: 100%;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
  white-space: nowrap;
  cursor: pointer;
}

.secret-permissions__button:disabled {
  color: var(--numen-hushed);
  cursor: default;
}

.secret-permissions__backdrop {
  position: fixed;
  inset: 0;
  z-index: 9;
}

.secret-permissions__list {
  position: absolute;
  inset-block-start: 100%;
  inset-inline-end: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-inline-size: 12rem;
  margin-block-start: 0.25rem;
  padding: 0.5rem 0.75rem;
  background-color: var(--numen-raised);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
}

.secret-permissions__declaration {
  color: var(--numen-ink);
}
</style>
