<script setup lang="ts">
/**
 * Placeholder displayed in place of a saved tab that could not be restored. The other saved tabs
 * open as usual.
 */
import { TriangleAlert } from '@lucide/vue'
import type { RestoreFailedState } from './restoreFailed'

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
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: 100%;
  block-size: 100%;
  padding: 2rem;
  box-sizing: border-box;
}

.tab-restore-failed__content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-inline-size: 28rem;
}

.tab-restore-failed__icon {
  inline-size: 2.5rem;
  block-size: 2.5rem;
  margin-block-end: 0.75rem;
  color: var(--numen-caution-fg);
}

.tab-restore-failed__title {
  margin: 0 0 0.5rem 0;
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
