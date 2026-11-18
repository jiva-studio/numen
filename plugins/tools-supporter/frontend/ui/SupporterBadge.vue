<script setup lang="ts">
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  isActive: boolean
}>()
</script>

<template>
  <span
    v-if="isActive"
    class="supporter-badge"
    role="status"
    :title="words.thankYou"
    :aria-label="words.thankYou"
  >
    <span class="supporter-badge__heart" aria-hidden="true">♥</span>
    <span class="supporter-badge__label">{{ words.badgeLabel }}</span>
  </span>
</template>
