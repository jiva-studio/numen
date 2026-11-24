<script setup lang="ts">
/**
 * The modal of the identity: the shared modal as a form with a title, a line that says what to do, the
 * body and a row of buttons. `step` moves the focus to the first control when it changes.
 */
import { ModalDialog } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ title: string; hint?: string; step?: string }>()

const emit = defineEmits<{
  (e: 'submit'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <ModalDialog
    is-form
    :label="title"
    :hint="hint ?? ''"
    :step="step ?? ''"
    @submit="emit('submit')"
    @cancel="emit('cancel')"
  >
    <slot />
    <template v-if="$slots['actions']" #actions><slot name="actions" /></template>
  </ModalDialog>
</template>
