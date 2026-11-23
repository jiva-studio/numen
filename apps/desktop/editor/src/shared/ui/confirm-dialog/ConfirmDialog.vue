<script setup lang="ts">
/**
 * A question over the window that is answered with one of two buttons, with a title and a text. Focus starts on the
 * cancel button; Escape, a press on the backdrop and the cancel button all emit `cancel`, and
 * Tab stays between the two buttons.
 */
import { onMounted, useId, useTemplateRef } from 'vue'
import { Button } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
withDefaults(
  defineProps<{
    text: string
    label: string
    confirmText: string
    cancelText: string
    tone?: 'danger' | 'default'
  }>(),
  { tone: 'default' },
)

const emit = defineEmits<{
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

/* ---------------------------------- State --------------------------------- */
const titleId = useId()
const textId = useId()
const dialog = useTemplateRef<HTMLElement>('dialog')
const cancel = useTemplateRef<InstanceType<typeof Button>>('cancel')

/* ---------------------------------- Hooks --------------------------------- */
onMounted(() => {
  const element = cancel.value?.$el
  if (element instanceof HTMLElement) element.focus()
})

/* -------------------------------- Handlers -------------------------------- */
function onKeepFocus(event: KeyboardEvent): void {
  const buttons = Array.from(dialog.value?.querySelectorAll<HTMLElement>('button') ?? [])
  const first = buttons[0]
  const last = buttons[buttons.length - 1]
  if (!first || !last) return
  if (event.shiftKey && event.target === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && event.target === last) {
    event.preventDefault()
    first.focus()
  }
}
</script>

<template>
  <div
    data-slot="confirm-dialog-backdrop"
    class="confirm-dialog-backdrop"
    @click.self="emit('cancel')"
    @keydown.esc="emit('cancel')"
    @keydown.tab="onKeepFocus"
  >
    <div
      ref="dialog"
      data-slot="confirm-dialog"
      role="alertdialog"
      aria-modal="true"
      :aria-labelledby="titleId"
      :aria-describedby="textId"
      :data-tone="tone"
      class="confirm-dialog"
    >
      <h2 :id="titleId" class="confirm-dialog__title">{{ label }}</h2>
      <p :id="textId" class="confirm-dialog__text">{{ text }}</p>
      <div class="confirm-dialog__actions">
        <Button :variant="tone === 'danger' ? 'danger' : 'solid'" @click="emit('confirm')">
          {{ confirmText }}
        </Button>
        <Button ref="cancel" variant="outline" @click="emit('cancel')">
          {{ cancelText }}
        </Button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.confirm-dialog-backdrop {
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--numen-gutter);
  background: color-mix(in oklab, var(--numen-surface), transparent 30%);
}

.confirm-dialog {
  --gap: 0.75rem;
  --max-width: 28rem;
  --pad: 1rem;
  display: flex;
  flex-direction: column;
  gap: var(--gap);
  max-inline-size: var(--max-width);
  padding: var(--pad);
  border: 1px solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
}

.confirm-dialog[data-tone='danger'] {
  border-color: var(--numen-alarm);
}

.confirm-dialog__text {
  margin: 0;
}

.confirm-dialog__title {
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 500;
}

.confirm-dialog__actions {
  --gap: 0.375rem;
  display: flex;
  justify-content: flex-end;
  gap: var(--gap);
}
</style>
