<script setup lang="ts">
/**
 * A question over the window that is answered with one of two buttons. Focus starts on the
 * cancel button; Escape, a press on the backdrop and the cancel button all emit `cancel`, and
 * Tab stays between the two buttons.
 */
import { onMounted, useId, useTemplateRef } from 'vue'
import Button from '../button/Button.vue'

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
    class="fixed inset-0 z-[var(--numen-lift-menu)] flex items-center justify-center bg-[color-mix(in_oklab,var(--numen-surface),transparent_30%)] p-[var(--numen-gutter)]"
    @click.self="emit('cancel')"
    @keydown.esc="emit('cancel')"
    @keydown.tab="onKeepFocus"
  >
    <div
      ref="dialog"
      data-slot="confirm-dialog"
      role="alertdialog"
      aria-modal="true"
      :aria-label="label"
      :aria-describedby="textId"
      :class="[
        'bg-raised text-ink rounded-node font-sans text-base',
        'flex max-w-md flex-col gap-3 border p-4',
        tone === 'danger' ? 'border-alarm' : 'border-field-rule',
      ]"
    >
      <p :id="textId" class="m-0">{{ text }}</p>
      <div class="flex justify-end gap-1.5">
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
