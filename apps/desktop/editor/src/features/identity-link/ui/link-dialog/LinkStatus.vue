<script setup lang="ts">
/** A state the dialog waits in or has reached: a spinner or a check above a title and the line that says what happens. */
import { Check } from '@lucide/vue'
import { Spinner } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ title: string; detail?: string; isDone?: boolean }>()
</script>

<template>
  <div class="link-status" role="status" :aria-busy="!isDone">
    <span v-if="isDone" class="link-status__done"><Check :size="18" aria-hidden="true" /></span>
    <Spinner v-else class="link-status__ring" />
    <p class="link-status__title">{{ title }}</p>
    <p v-if="detail" class="link-status__detail">{{ detail }}</p>
    <slot />
  </div>
</template>

<style scoped>
.link-status {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--numen-inset);
  padding-block: 1rem;
  text-align: center;
}

.link-status__ring {
  --spinner-size: 1.75rem;
  --spinner-thickness: 0.15rem;
  color: var(--numen-accent);
}

.link-status__done {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
  border-radius: 50%;
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}

.link-status__title {
  margin: 0;
  color: var(--numen-ink);
  font-weight: 600;
}

.link-status__detail {
  margin: 0;
  color: var(--numen-hushed);
}
</style>
