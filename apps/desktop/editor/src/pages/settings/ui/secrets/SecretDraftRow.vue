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
  /** Creates the secret and resolves true when it was created; the field then keeps a failed value. */
  submit?: ((value: string) => Promise<boolean>) | undefined
}>()

const emit = defineEmits<{
  (e: 'create', value: string): void
  (e: 'discard'): void
}>()
</script>

<template>
  <li class="secret-draft">
    <span class="secret-draft__name">{{ name }}</span>
    <SecretValueInput
      :label="words.secretValue(name)"
      :placeholder="words.keyPlaceholder"
      :submit="submit"
      @commit="(value) => emit('create', value)"
    />
    <span />
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
  </li>
</template>

<style scoped>
.secret-draft {
  display: grid;
  grid-template-columns: 8rem minmax(0, 1fr) 9rem 5.5rem;
  align-items: center;
  gap: 0.5rem var(--numen-panel-gap);
  padding-block: 0.5rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  list-style: none;
}

.secret-draft__name {
  padding-inline: 0.625rem;
  overflow: hidden;
  color: var(--numen-ink);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.secret-draft__actions {
  min-inline-size: 1.75rem;
}
</style>
