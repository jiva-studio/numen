<script setup lang="ts">
/** The two ways to link: show a QR code, or read one. */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'

/* --------------------------------- Emits ---------------------------------- */
const emit = defineEmits<{
  (e: 'show'): void
  (e: 'scan'): void
}>()
</script>

<template>
  <div class="link-choices">
    <Button type="button" variant="outline" @click="emit('show')">{{ words.linkShow }}</Button>
    <Button type="button" variant="outline" @click="emit('scan')">{{ words.linkScan }}</Button>
  </div>
</template>

<style scoped>
.link-choices {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
