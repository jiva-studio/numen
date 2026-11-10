<script setup lang="ts">
/**
 * Placeholder displayed in place of a saved tab that could not be restored. The other saved tabs
 * open as usual.
 */
import { TriangleAlert } from '@lucide/vue'
import type { RestoreFailedState } from '../model/restoreFailed'

defineProps<{
  state: RestoreFailedState
}>()
</script>

<template>
  <div class="tab-restore-failed" role="alert">
    <div class="tab-restore-failed__content">
      <TriangleAlert class="tab-restore-failed__icon" aria-hidden="true" />
      <h3 class="tab-restore-failed__title">Could not restore this tab</h3>
      <p class="tab-restore-failed__message">{{ state.message }}</p>
    </div>
  </div>
</template>

<style scoped>
.tab-restore-failed {
  --pad: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: 100%;
  block-size: 100%;
  padding: var(--pad);
  box-sizing: border-box;
}

.tab-restore-failed__content {
  --max-width: 28rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-inline-size: var(--max-width);
}

.tab-restore-failed__icon {
  --size: 2.5rem;
  --margin-end: 0.75rem;
  inline-size: var(--size);
  block-size: var(--size);
  margin-block-end: var(--margin-end);
  color: var(--numen-caution-fg);
}

.tab-restore-failed__title {
  --margin: 0 0 0.5rem 0;
  margin: var(--margin);
  font-size: var(--numen-text-4);
  font-weight: 600;
  color: var(--numen-ink);
}

.tab-restore-failed__message {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  line-height: 1.4;
}
</style>
