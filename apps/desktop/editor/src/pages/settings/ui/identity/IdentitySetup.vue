<script setup lang="ts">
/**
 * The setup of a new identity: the twelve recovery words, the check of three of them, and the
 * choice to keep the identity or drop the setup.
 */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import ConfirmWords from './ConfirmWords.vue'
import RecoveryWords from './RecoveryWords.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  recoveryWords: readonly string[]
  positions: readonly number[]
  entered: Readonly<Record<number, string>>
  isBusy: boolean
}>()

const emit = defineEmits<{
  (e: 'change', position: number, value: string): void
  (e: 'keep'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <RecoveryWords :recovery-words="recoveryWords" />
  <ConfirmWords
    :positions="positions"
    :entered="entered"
    :is-disabled="isBusy"
    @change="(position, value) => emit('change', position, value)"
    @submit="emit('keep')"
  />
  <div class="identity-setup__actions">
    <Button type="button" variant="solid" :disabled="isBusy" @click="emit('keep')">
      {{ isBusy ? words.identityKeeping : words.identityKeep }}
    </Button>
    <Button type="button" variant="outline" :disabled="isBusy" @click="emit('cancel')">
      {{ words.identityCancel }}
    </Button>
  </div>
</template>

<style scoped>
.identity-setup__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
