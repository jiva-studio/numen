<script setup lang="ts">
/**
 * The restore of an identity from two sheets of a recovery kit typed by the person.
 */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import SheetsInput from './SheetsInput.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ texts: readonly [string, string]; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'change', index: number, value: string): void
  (e: 'restore'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <SheetsInput
    :texts="texts"
    :is-disabled="isBusy"
    @change="(index, value) => emit('change', index, value)"
  />
  <div class="identity-restore-sheets__actions">
    <Button type="button" variant="solid" :disabled="isBusy" @click="emit('restore')">
      {{ isBusy ? words.identityRestoring : words.identityRestore }}
    </Button>
    <Button type="button" variant="outline" :disabled="isBusy" @click="emit('cancel')">
      {{ words.identityCancel }}
    </Button>
  </div>
</template>

<style scoped>
.identity-restore-sheets__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
