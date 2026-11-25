<script setup lang="ts">
/**
 * The modal of the identity: the shared modal as a form with a title, a line that says what to do, the
 * body and a row of buttons. `step` moves the focus to the first control when it changes.
 */
import { Dialog } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ title: string; hint?: string; caution?: string; step?: string }>()

const emit = defineEmits<{
  (e: 'submit'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <Dialog
    is-form
    :label="title"
    :hint="hint ?? ''"
    :step="step ?? ''"
    @submit="emit('submit')"
    @cancel="emit('cancel')"
  >
    <p v-if="caution" class="identity-dialog__caution" role="note">{{ caution }}</p>
    <slot />
    <template v-if="$slots['actions']" #actions><slot name="actions" /></template>
  </Dialog>
</template>

<style scoped>
.identity-dialog__caution {
  margin: 0;
  color: var(--numen-hushed);
}
</style>
