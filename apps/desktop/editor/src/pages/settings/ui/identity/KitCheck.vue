<script setup lang="ts">
/**
 * The check of two sheets against the account of this device: the two fields, the button, and the
 * confirmation when the sheets fit.
 */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import SheetsInput from './SheetsInput.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ texts: readonly [string, string]; isFitting: boolean; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'change', index: number, value: string): void
  (e: 'check'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <div class="kit-check">
    <SheetsInput
      :texts="texts"
      :is-disabled="isBusy"
      @change="(index, value) => emit('change', index, value)"
    />
    <p v-if="isFitting" role="status" class="kit-check__fits">{{ words.kitFits }}</p>
    <div class="kit-check__actions">
      <Button type="button" variant="solid" :disabled="isBusy" @click="emit('check')">
        {{ words.kitCheckAction }}
      </Button>
      <Button type="button" variant="outline" :disabled="isBusy" @click="emit('cancel')">
        {{ words.identityCancel }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.kit-check {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.kit-check__fits {
  margin: 0;
  font-size: var(--numen-text-1);
}

.kit-check__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
