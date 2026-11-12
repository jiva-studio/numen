<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
withDefaults(
  defineProps<{
    tool: string
    subject?: string
    isWorking?: boolean
  }>(),
  { subject: '', isWorking: false },
)
</script>

<template>
  <p class="tool-call numen text-hushed flex items-center gap-2 font-sans text-base">
    <span class="tool-call__mark" :data-working="isWorking || undefined" />
    <span class="min-w-0 truncate">{{ tool }}</span>
    <span v-if="subject" class="min-w-0 flex-1 truncate opacity-70">{{ subject }}</span>
  </p>
</template>

<style scoped>
.tool-call__mark {
  --mark-size: 0.45em;
  flex: none;
  inline-size: var(--mark-size);
  block-size: var(--mark-size);
  border-radius: var(--numen-radius-pill);
  background: currentColor;
  opacity: 0.4;
  transition:
    opacity 0.2s,
    transform 0.2s;
}

.tool-call__mark[data-working] {
  opacity: 1;
  animation: tool-call-pulse 1.4s ease-in-out infinite;
}

@keyframes tool-call-pulse {
  0%,
  100% {
    transform: scale(0.85);
    opacity: 0.4;
  }
  50% {
    transform: scale(1.15);
    opacity: 1;
  }
}
</style>
