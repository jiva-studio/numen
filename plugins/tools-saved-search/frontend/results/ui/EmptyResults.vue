<script setup lang="ts">
/** The results area with no result in it: why, in words, and the button that brings the results back. */
import { Button } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  text: string
  actionText?: string
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'act'): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onAct() {
  emit('act')
}
</script>

<template>
  <div class="saved-search__empty" role="status">
    <p class="saved-search__empty-text">{{ text }}</p>
    <Button v-if="actionText" variant="outline" size="small" @click="onAct">
      {{ actionText }}
    </Button>
  </div>
</template>
