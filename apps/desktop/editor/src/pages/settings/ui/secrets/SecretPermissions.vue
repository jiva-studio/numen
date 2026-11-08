<script setup lang="ts">
/**
 * The plugin declarations that may read a secret: a button that reads the number of plugins and
 * opens a checklist with a check box for each declaration, labelled with the plugin title.
 */
import { computed, ref } from 'vue'
import { ChevronDown } from '@lucide/vue'
import type { Secret } from '@/entities/settings'
import { isAllowed, type Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  secret: Secret
  declarations: readonly Declaration[]
}>()

const emit = defineEmits<{
  (e: 'toggle', declaration: Declaration, isAllowed: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const count = computed(
  () => props.declarations.filter((one) => isAllowed(props.secret, one)).length,
)

/* -------------------------------- Handlers -------------------------------- */
function onToggle(declaration: Declaration, event: Event): void {
  emit('toggle', declaration, (event.target as HTMLInputElement).checked)
}
</script>

<template>
  <div class="secret-permissions" @keydown.esc="isOpen = false">
    <button
      type="button"
      class="secret-permissions__button"
      :disabled="declarations.length === 0"
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
      <label
        v-for="one in declarations"
        :key="`${one.pluginId}/${one.name}/${one.group}`"
        class="secret-permissions__declaration"
        :title="one.description"
      >
        <input
          type="checkbox"
          :checked="isAllowed(secret, one)"
          @change="(event) => onToggle(one, event)"
        />
        {{ one.pluginName }}
      </label>
    </div>
  </div>
</template>

<style scoped>
.secret-permissions {
  position: relative;
}

.secret-permissions__button {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
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
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--numen-ink);
  cursor: pointer;
}
</style>
