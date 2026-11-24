<script setup lang="ts">
/**
 * Where the keys are kept: the system keychain, or on a computer without one the file and the way it
 * is protected. The action to change it or to enter the passphrase sits in the header of the card.
 */
import { computed } from 'vue'
import { IslandCard } from '@numen/ui'
import type { KeyFileState } from '@/entities/key-file'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: KeyFileState }>()

const emit = defineEmits<{ (e: 'change'): void; (e: 'unlock'): void }>()

/* --------------------------------- State ---------------------------------- */
const detail = computed(() => {
  if (!props.state.isInUse) return words.rowKeychain
  if (props.state.mode === 'plaintext') return words.rowPlaintext
  if (props.state.mode === 'passphrase') {
    return props.state.isLocked ? words.rowLocked : words.rowPassphrase
  }
  return words.rowUnset
})
</script>

<template>
  <IslandCard :title="words.rowTitle">
    <template v-if="state.isInUse" #aside>
      <button
        v-if="state.isLocked"
        type="button"
        class="key-file-row__action"
        @click="emit('unlock')"
      >
        {{ words.rowUnlock }}
      </button>
      <button v-else type="button" class="key-file-row__action" @click="emit('change')">
        {{ words.rowChange }}
      </button>
    </template>
    <p class="key-file-row__detail">{{ detail }}</p>
  </IslandCard>
</template>

<style scoped>
.key-file-row__detail {
  margin: 0;
  padding-block: calc(var(--numen-card-pad) - var(--numen-card-inset));
  color: var(--numen-hushed);
}

.key-file-row__action {
  margin: 0;
  padding: 0 0.375rem;
  border: none;
  background: none;
  color: var(--numen-accent);
  font: inherit;
  font-size: var(--numen-text-1);
  font-weight: 500;
  cursor: pointer;
}
</style>
