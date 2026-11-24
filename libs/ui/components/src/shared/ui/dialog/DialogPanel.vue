<script setup lang="ts">
/** The backdrop and the panel of one open dialog: its markup, the focus inside it and the keys that cancel it. */
import { onBeforeUnmount, onMounted, useId, useTemplateRef, watch } from 'vue'
import { doc } from '@/shared/api/dom'
import type { DialogEvents, DialogPanelProps } from './Dialog.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<DialogPanelProps>(), {
  hint: '',
  role: 'dialog',
  tone: 'default',
  describedBy: '',
  step: '',
  size: 'md',
  isForm: false,
})

const emit = defineEmits<DialogEvents>()

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
    data-slot="dialog-backdrop"
    class="dialog-backdrop"
    :data-scope="scope"
    @click.self="emit('cancel')"
    @keydown.esc="emit('cancel')"
    @keydown.tab="onKeepFocus"
  >
    <component
      :is="isForm ? 'form' : 'div'"
      ref="panel"
      data-slot="dialog"
      class="dialog"
      :role="role"
      aria-modal="true"
      :aria-labelledby="titleId"
      :aria-describedby="describedBy === '' ? undefined : describedBy"
      :data-tone="tone"
      :data-size="size"
      @submit.prevent="emit('submit')"
    >
      <header class="dialog__head">
        <h2 :id="titleId" class="dialog__title">{{ label }}</h2>
        <p v-if="hint !== ''" class="dialog__hint">{{ hint }}</p>
      </header>
      <div class="dialog__body"><slot /></div>
      <div v-if="$slots['actions']" class="dialog__actions"><slot name="actions" /></div>
    </component>
  </div>
</template>

<style scoped>
.dialog-backdrop {
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in oklab, var(--numen-surface), transparent 30%);
}

.dialog-backdrop[data-scope='host'] {
  position: absolute;
}

.dialog[data-size='sm'] {
  --width: 22rem;
}

.dialog[data-size='md'] {
  --width: 28rem;
}

.dialog[data-size='lg'] {
  --width: 36rem;
}

.dialog {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  box-sizing: border-box;
  inline-size: min(calc(100% - 2rem), var(--width));
  max-block-size: calc(100% - 2rem);
  overflow: hidden;
  padding: var(--numen-gutter);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  line-height: var(--numen-line-height);
}

.dialog[data-tone='danger'] {
  border-color: var(--numen-alarm);
}

.dialog__head {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.dialog__title {
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 500;
}

.dialog__hint {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.dialog__head,
.dialog__actions {
  flex: none;
}

.dialog__body {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 0.5rem;
  min-block-size: 0;
  margin: -0.25rem;
  padding: 0.25rem;
  overflow-y: auto;
}

.dialog__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>
