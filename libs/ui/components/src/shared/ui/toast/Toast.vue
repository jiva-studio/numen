<script setup lang="ts">
/**
 * A short message at the bottom of the window that goes away by itself: it says that something the
 * person did is done. An empty text shows nothing. Each new text is shown for the whole duration.
 */
import { useExpiringText } from '@/shared/model/useExpiringText'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<{ text: string; durationMs?: number }>(), {
  durationMs: 4000,
})

/* --------------------------------- State ---------------------------------- */
const shown = useExpiringText(
  () => props.text,
  () => props.durationMs,
)
</script>

<template>
  <Teleport to="body">
    <div class="toast-region numen" role="status" aria-live="polite">
      <p v-if="shown !== ''" :key="shown" class="toast">{{ shown }}</p>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-region {
  position: fixed;
  inset-block-end: 1.5rem;
  inset-inline: 0;
  z-index: var(--numen-lift-tooltip);
  display: flex;
  justify-content: center;
  pointer-events: none;
}

.toast {
  box-sizing: border-box;
  max-inline-size: min(28rem, calc(100vw - 2rem));
  margin: 0;
  padding: 0.5rem 0.875rem;
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
  color: var(--numen-ink);
  box-shadow: var(--numen-shadow-card);
  font-size: var(--numen-text-1);
  animation: toast-show var(--numen-motion) var(--numen-easing);
}

@keyframes toast-show {
  from {
    opacity: 0;
    transform: translateY(0.5rem);
  }
}

@media (prefers-reduced-motion: reduce) {
  .toast {
    animation: none;
  }
}
</style>
