<script setup lang="ts">
/**
 * The one modal of the interface: a dimmed backdrop over the tab it is drawn in and a panel in its middle with a title,
 * a hint, the body and a row of buttons at the right. The panel is 28rem wide and as high as its content. Escape and a
 * press on the backdrop emit `cancel`, Tab stays inside the panel, the first control takes the focus when the panel opens
 * and each time `step` changes, and the focus returns to the control that had it before the panel opened.
 */
import { onBeforeUnmount, onMounted, useId, useTemplateRef, watch } from 'vue'
import { doc } from '@/shared/api/dom'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    label: string
    hint?: string
    role?: 'dialog' | 'alertdialog'
    tone?: 'danger' | 'default'
    describedBy?: string
    step?: string
    isForm?: boolean
  }>(),
  { hint: '', role: 'dialog', tone: 'default', describedBy: '', step: '', isForm: false },
)

const emit = defineEmits<{
  (e: 'submit'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const CONTROLS = 'input, textarea, select, button, [href], [role="combobox"], [role="switch"]'
const titleId = useId()
const panel = useTemplateRef<HTMLElement>('panel')
const active = doc.activeElement
const opener = active instanceof HTMLElement ? active : null

/* ---------------------------------- Hooks --------------------------------- */
onMounted(focusFirst)
watch(() => props.step, focusFirst, { flush: 'post' })
onBeforeUnmount(() => {
  if (opener?.isConnected) opener.focus()
})

/* -------------------------------- Handlers -------------------------------- */
function onKeepFocus(event: KeyboardEvent): void {
  const items = getControls()
  const first = items[0]
  const last = items[items.length - 1]
  if (!first || !last) return
  if (event.shiftKey && event.target === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && event.target === last) {
    event.preventDefault()
    first.focus()
  }
}

/* -------------------------------- Helpers --------------------------------- */
function getControls(): HTMLElement[] {
  return Array.from(panel.value?.querySelectorAll<HTMLElement>(CONTROLS) ?? []).filter(
    (one) => !one.hasAttribute('disabled'),
  )
}

function focusFirst(): void {
  getControls()[0]?.focus()
}
</script>

<template>
  <div
    data-slot="modal-dialog-backdrop"
    class="modal-dialog-backdrop"
    @click.self="emit('cancel')"
    @keydown.esc="emit('cancel')"
    @keydown.tab="onKeepFocus"
  >
    <component
      :is="isForm ? 'form' : 'div'"
      ref="panel"
      data-slot="modal-dialog"
      class="modal-dialog"
      :role="role"
      aria-modal="true"
      :aria-labelledby="titleId"
      :aria-describedby="describedBy === '' ? undefined : describedBy"
      :data-tone="tone"
      @submit.prevent="emit('submit')"
    >
      <header class="modal-dialog__head">
        <h2 :id="titleId" class="modal-dialog__title">{{ label }}</h2>
        <p v-if="hint !== ''" class="modal-dialog__hint">{{ hint }}</p>
      </header>
      <div class="modal-dialog__body"><slot /></div>
      <div v-if="$slots['actions']" class="modal-dialog__actions"><slot name="actions" /></div>
    </component>
  </div>
</template>

<style scoped>
.modal-dialog-backdrop {
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--numen-gutter);
  background: color-mix(in oklab, var(--numen-surface), transparent 30%);
}

.modal-dialog {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  box-sizing: border-box;
  inline-size: min(100%, 28rem);
  max-block-size: 100%;
  overflow-y: auto;
  padding: var(--numen-gutter);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  line-height: var(--numen-line-height);
}

.modal-dialog[data-tone='danger'] {
  border-color: var(--numen-alarm);
}

.modal-dialog__head {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.modal-dialog__title {
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 500;
}

.modal-dialog__hint {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.modal-dialog__body {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.modal-dialog__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>
