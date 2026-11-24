<script setup lang="ts">
/**
 * Where the keys are kept: the system keychain, or on a computer without one the file and the way it
 * is protected. The action to change it or to enter the passphrase sits in the header of the card.
 */
import { computed } from 'vue'
import { LockOpen, Pencil } from '@lucide/vue'
import { CardAction, IslandCard } from '@numen/ui'
import type { KeyFileState } from '@/entities/key-file'
import { describeKeyFile } from '../lib/describeKeyFile'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: KeyFileState }>()

const emit = defineEmits<{ (e: 'change'): void; (e: 'unlock'): void }>()

/* --------------------------------- State ---------------------------------- */
const detail = computed(() => describeKeyFile(props.state))
</script>

<template>
  <IslandCard :title="words.rowTitle">
    <template v-if="state.isInUse" #aside>
      <CardAction v-if="state.isLocked" is-primary :label="words.rowUnlock" @click="emit('unlock')">
        <LockOpen :size="14" aria-hidden="true" />
      </CardAction>
      <CardAction v-else is-primary :label="words.rowChange" @click="emit('change')">
        <Pencil :size="14" aria-hidden="true" />
      </CardAction>
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
</style>
