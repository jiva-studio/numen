<script setup lang="ts">
/**
 * What the workspace shows in place of its tabs when the window could not load what it opens with.
 */
import { TriangleAlert } from '@lucide/vue'
import { Button } from '@numen/ui'

defineProps<{
  message: string
}>()

defineEmits<{
  retry: []
}>()
</script>

<template>
  <div class="bootstrap-failure" role="alert">
    <TriangleAlert class="bootstrap-failure__icon" aria-hidden="true" />
    <p class="bootstrap-failure__message">Could not load the workspace: {{ message }}.</p>
    <Button @click="$emit('retry')">Retry</Button>
  </div>
</template>

<style scoped>
.bootstrap-failure {
  --gap: 0.75rem;
  --pad: 2rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--gap);
  inline-size: 100%;
  block-size: 100%;
  padding: var(--pad);
  box-sizing: border-box;
  text-align: center;
}

.bootstrap-failure__icon {
  --size: 2.5rem;
  inline-size: var(--size);
  block-size: var(--size);
  color: var(--numen-caution-fg);
}

.bootstrap-failure__message {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}
</style>
