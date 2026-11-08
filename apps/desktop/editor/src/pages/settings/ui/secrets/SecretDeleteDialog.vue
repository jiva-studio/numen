<script setup lang="ts">
/**
 * The confirmation before a secret or a group is deleted, with the question it asks as text. The
 * keyboard starts on Cancel.
 */
import { onMounted, useTemplateRef } from 'vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
withDefaults(defineProps<{ text: string; label?: string }>(), { label: words.secretDelete })

const emit = defineEmits<{
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const cancel = useTemplateRef<InstanceType<typeof Button>>('cancel')

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  const element = cancel.value?.$el
  if (element instanceof HTMLElement) element.focus()
})
</script>

<template>
  <div class="secret-dialog__backdrop" @keydown.esc="emit('cancel')">
    <div class="secret-dialog" role="alertdialog" aria-modal="true" :aria-label="label">
      <p class="secret-dialog__text">{{ text }}</p>
      <div class="secret-dialog__answers">
        <Button variant="danger" size="small" @click="emit('confirm')">
          {{ words.secretDeleteConfirm }}
        </Button>
        <Button ref="cancel" variant="outline" size="small" @click="emit('cancel')">
          {{ words.secretDeleteCancel }}
        </Button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.secret-dialog__backdrop {
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--numen-gutter);
  background-color: color-mix(in oklab, var(--numen-surface), transparent 30%);
}

.secret-dialog {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-inline-size: 28rem;
  padding: 1rem;
  background-color: var(--numen-raised);
  border: var(--numen-stroke) solid var(--numen-alarm);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
}

.secret-dialog__text {
  margin: 0;
}

.secret-dialog__answers {
  display: flex;
  justify-content: flex-end;
  gap: 0.375rem;
}
</style>
