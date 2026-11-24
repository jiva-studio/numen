<script setup lang="ts">
/**
 * A fingerprint in monospace. In a narrow place the middle is cut with an ellipsis and the last two
 * groups stay, so two fingerprints can still be told apart; where there is room it reads in full.
 * The whole value is the title, the label for a screen reader and what a selection copies.
 */
import { computed } from 'vue'
import { splitFingerprint } from './split'

const props = defineProps<{ value: string }>()

const parts = computed(() => splitFingerprint(props.value))
</script>

<template>
  <span class="fingerprint" role="group" :title="value" :aria-label="value">
    <span class="fingerprint__head" aria-hidden="true">{{ parts.head }}</span>
    <span class="fingerprint__tail" aria-hidden="true">{{ parts.tail }}</span>
  </span>
</template>

<style scoped>
.fingerprint {
  display: flex;
  min-inline-size: 0;
  font-family: var(--numen-font-mono);
  user-select: text;
}

.fingerprint__head {
  flex: 0 1 auto;
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* The space between the head and the tail belongs to the tail, so a copied value keeps it. */
.fingerprint__tail {
  flex: none;
  white-space: pre;
}
</style>
