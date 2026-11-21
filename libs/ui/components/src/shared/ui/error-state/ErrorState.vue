<script setup lang="ts">
/** The state of a pane that cannot show its content: an icon, an optional heading and the message. */
import { TriangleAlert } from '@lucide/vue'

defineProps<{ message: string; heading?: string }>()
</script>

<template>
  <div class="error-state numen" role="alert">
    <div class="error-state__content">
      <TriangleAlert class="error-state__icon" aria-hidden="true" />
      <h3 v-if="heading" class="error-state__heading">{{ heading }}</h3>
      <p class="error-state__message">{{ message }}</p>
      <slot />
    </div>
  </div>
</template>

<style scoped>
.error-state {
  --pad: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: 100%;
  block-size: 100%;
  padding: var(--pad);
  box-sizing: border-box;
}

.error-state__content {
  --max-width: 28rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-inline-size: var(--max-width);
}

.error-state__icon {
  --size: 2.5rem;
  --margin-end: 0.75rem;
  inline-size: var(--size);
  block-size: var(--size);
  margin-block-end: var(--margin-end);
  color: var(--numen-caution-fg);
}

.error-state__heading {
  --margin: 0 0 0.5rem 0;
  margin: var(--margin);
  font-size: var(--numen-text-4);
  font-weight: 600;
  color: var(--numen-ink);
}

.error-state__message {
  --margin: 0 0 0.75rem 0;
  margin: var(--margin);
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  line-height: 1.4;
  overflow-wrap: anywhere;
}
</style>
