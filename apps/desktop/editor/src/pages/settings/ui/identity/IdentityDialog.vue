<script setup lang="ts">
/**
 * The modal dialog of the identity card: a titled form over a backdrop. Escape and a press on the
 * backdrop cancel, Tab stays inside the dialog, and the first field or button takes the focus when
 * the dialog opens and each time `step` changes.
 */
import { onMounted, useId, useTemplateRef, watch } from 'vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ title: string; step?: string }>()

const emit = defineEmits<{
  (e: 'submit'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const titleId = useId()
const dialog = useTemplateRef<HTMLElement>('dialog')

/* ---------------------------------- Hooks --------------------------------- */
onMounted(focusFirst)
watch(() => props.step, focusFirst, { flush: 'post' })

/* -------------------------------- Handlers -------------------------------- */
function onKeepFocus(event: KeyboardEvent): void {
  const items = Array.from(
    dialog.value?.querySelectorAll<HTMLElement>('input, button') ?? [],
  ).filter((one) => !one.hasAttribute('disabled'))
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
function focusFirst(): void {
  dialog.value?.querySelector<HTMLElement>('input, button')?.focus()
}
</script>

<template>
  <div
    class="identity-dialog__backdrop"
    @click.self="emit('cancel')"
    @keydown.esc="emit('cancel')"
    @keydown.tab="onKeepFocus"
  >
    <form
      ref="dialog"
      class="identity-dialog"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="titleId"
      @submit.prevent="emit('submit')"
    >
      <h2 :id="titleId" class="identity-dialog__title">{{ title }}</h2>
      <slot />
    </form>
  </div>
</template>

<style scoped>
.identity-dialog__backdrop {
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--numen-gutter);
  background: color-mix(in oklab, var(--numen-surface), transparent 30%);
}

.identity-dialog {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  box-sizing: border-box;
  inline-size: 100%;
  max-inline-size: 28rem;
  max-block-size: 100%;
  overflow-y: auto;
  padding: 1rem 1.25rem;
  background: var(--numen-raised);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  line-height: var(--numen-line-height);
}

.identity-dialog__title {
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 500;
}
</style>
