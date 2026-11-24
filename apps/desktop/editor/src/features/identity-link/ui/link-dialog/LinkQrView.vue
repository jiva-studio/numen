<script setup lang="ts">
/** The QR code of this device. Its box keeps its size from the first frame and holds a spinner until the text arrives. */
import { QrCode, Spinner } from '@numen/ui'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ qrText: string }>()

/* --------------------------------- State ---------------------------------- */
const QR_SIZE = 200
</script>

<template>
  <div class="link-qr" :style="{ '--qr-size': `${QR_SIZE}px` }">
    <QrCode v-if="qrText !== ''" :text="qrText" :label="words.linkQr" :size="QR_SIZE" />
    <Spinner v-else />
  </div>
</template>

<style scoped>
.link-qr {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  inline-size: var(--qr-size);
  block-size: var(--qr-size);
  margin-inline: auto;
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
}
</style>
