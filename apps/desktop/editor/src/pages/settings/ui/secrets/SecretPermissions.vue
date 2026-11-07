<script setup lang="ts">
/**
 * The plugin declarations that may be allowed to read a secret, each a check box labelled with the
 * plugin and the description the plugin gave the declaration.
 */
import type { Secret } from '@/entities/settings'
import { isAllowed, type Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  secret: Secret
  declarations: readonly Declaration[]
}>()

const emit = defineEmits<{
  (e: 'toggle', declaration: Declaration, isAllowed: boolean): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onToggle(declaration: Declaration, event: Event): void {
  emit('toggle', declaration, (event.target as HTMLInputElement).checked)
}
</script>

<template>
  <ul v-if="declarations.length > 0" class="secret-permissions" :aria-label="words.secretAllowedTo">
    <li v-for="one in declarations" :key="`${one.pluginId}/${one.name}/${one.group}`">
      <label class="secret-permissions__declaration">
        <input
          type="checkbox"
          :checked="isAllowed(secret, one)"
          @change="(event) => onToggle(one, event)"
        />
        <span class="secret-permissions__plugin">{{ one.pluginName }}</span>
        <span class="secret-permissions__description">{{ one.description || one.title }}</span>
      </label>
    </li>
  </ul>
</template>

<style scoped>
.secret-permissions {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.secret-permissions__declaration {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  cursor: pointer;
}

.secret-permissions__plugin {
  color: var(--numen-ink);
}

.secret-permissions__description {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
