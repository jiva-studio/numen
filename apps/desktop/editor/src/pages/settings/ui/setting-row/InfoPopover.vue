<script setup lang="ts">
/**
 * The details of a setting over the page, outside the card, placed against the info button that
 * opened it. It takes the focus when drawn. Escape asks to close it and give the focus back to the
 * button; a press outside, a scroll or a resize of the page asks to close it without.
 */
import { onMounted, useTemplateRef } from 'vue'
import { useMenuDismissal } from '@numen/ui'
import { WORDS as words } from '../../words'
import { useInfoPopoverPlace } from './useInfoPopoverPlace'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  name: string
  text: string
  anchor: HTMLElement | null
  from: HTMLElement | null
}>()

const emit = defineEmits<{
  (e: 'close', isFocusReturned: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const popover = useTemplateRef<HTMLElement>('popover')
const { style, place } = useInfoPopoverPlace()
const dismissal = useMenuDismissal(
  popover,
  () => emit('close', false),
  undefined,
  () => props.from,
)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  dismissal.listen()
  place(props.anchor, popover.value)
  popover.value?.focus()
})

/* -------------------------------- Handlers -------------------------------- */
function onEscape(): void {
  emit('close', true)
}
</script>

<template>
  <Teleport to="body">
    <div
      ref="popover"
      class="settings__popover"
      role="dialog"
      :style="style"
      :aria-label="words.aboutSetting(props.name)"
      tabindex="-1"
      @keydown.esc.stop.prevent="onEscape"
    >
      <p class="settings__popover-title">{{ props.name }}</p>
      <p class="settings__popover-text">{{ props.text }}</p>
    </div>
  </Teleport>
</template>

<style scoped>
.settings__popover {
  position: fixed;
  z-index: var(--numen-lift-menu);
  box-sizing: border-box;
  inline-size: max-content;
  max-inline-size: min(26rem, calc(100vw - 1rem));
  padding: 0.75rem 0.875rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-popup);
  background-color: var(--numen-raised);
  box-shadow: var(--numen-shadow-card);
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
  line-height: 1.5;
}

.settings__popover:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.settings__popover-title {
  margin: 0 0 0.25rem;
  font-weight: 600;
}

.settings__popover-text {
  margin: 0;
  color: var(--numen-hushed);
}
</style>
