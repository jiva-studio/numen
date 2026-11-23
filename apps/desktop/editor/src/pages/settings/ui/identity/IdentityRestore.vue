<script setup lang="ts">
/**
 * The restore of an identity from its twelve words typed by the person.
 */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import RestoreWords from './RestoreWords.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ text: string; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'change', value: string): void
  (e: 'restore'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <RestoreWords :text="text" :is-disabled="isBusy" @change="(value) => emit('change', value)" />
  <div class="identity-restore__actions">
    <Button type="button" variant="solid" :disabled="isBusy" @click="emit('restore')">
      {{ isBusy ? words.identityRestoring : words.identityRestore }}
    </Button>
    <Button type="button" variant="outline" :disabled="isBusy" @click="emit('cancel')">
      {{ words.identityCancel }}
    </Button>
  </div>
</template>

<style scoped>
.identity-restore__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
