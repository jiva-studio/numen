<script setup lang="ts">
/**
 * A secret that does not exist yet: its name and a masked field. The secret is created when a
 * value is entered. The empty secret of a group a plugin asks for has no way to be discarded.
 */
import { X } from '@lucide/vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import SecretValueInput from './SecretValueInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  name: string
  canDiscard: boolean
}>()

const emit = defineEmits<{
  (e: 'create', value: string): void
  (e: 'discard'): void
}>()
</script>

<template>
  <li class="secret-draft">
    <div class="secret-draft__title">
      <span class="secret-draft__name">{{ name }}</span>
    </div>
    <div class="secret-draft__actions">
      <Button
        v-if="canDiscard"
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretDiscard"
        :title="words.secretDiscard"
        @click="emit('discard')"
      >
        <X :size="14" aria-hidden="true" />
      </Button>
    </div>
    <SecretValueInput
      class="secret-draft__value"
      :label="words.secretValue(name)"
      :placeholder="words.keyPlaceholder"
      @commit="(value) => emit('create', value)"
    />
  </li>
</template>

<style scoped>
.secret-draft {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.5rem var(--numen-panel-gap);
  padding-block: 0.75rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  list-style: none;
}

.secret-draft__name {
  color: var(--numen-ink);
  font-weight: 500;
}

.secret-draft__value {
  grid-column: 1 / -1;
}
</style>
