<script setup lang="ts">
/**
 * The buttons at the foot of the connection dialog: delete, for a stored connection, cancel and
 * save. Save submits the form the buttons are in.
 */
import { Trash2 } from '@lucide/vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  canDelete: boolean
  canSave: boolean
}>()

const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'delete'): void
}>()
</script>

<template>
  <div class="connection-actions">
    <Button
      v-if="props.canDelete"
      type="button"
      variant="outline"
      class="connection-actions__delete"
      :aria-label="words.connectionMenuDelete"
      :title="words.connectionMenuDelete"
      @click="emit('delete')"
    >
      <Trash2 :size="14" aria-hidden="true" />
      {{ words.connectionMenuDelete }}
    </Button>
    <Button type="button" variant="outline" @click="emit('cancel')">
      {{ words.connectionCancel }}
    </Button>
    <Button type="submit" variant="solid" :disabled="!props.canSave">
      {{ words.connectionSave }}
    </Button>
  </div>
</template>

<style scoped>
.connection-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.375rem;
}

.connection-actions__delete {
  margin-inline-end: auto;
}
</style>
