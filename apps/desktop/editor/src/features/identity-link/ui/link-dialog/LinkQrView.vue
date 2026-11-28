<script setup lang="ts">
/** The QR code of this device. Its box keeps its size from the first frame and holds a spinner until the text arrives. */
import { QrCode, Spinner } from '@numen/ui'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ qrText: string }>()
</script>

<template>
  <div class="link-qr">
    <QrCode v-if="qrText !== ''" class="link-qr__code" :text="qrText" :label="words.linkQr" />
    <Spinner v-else />
  </div>
</template>

<style scoped>
.link-qr {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  inline-size: 12.5rem;
  block-size: 12.5rem;
  margin-inline: auto;
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
}

.link-qr__code {
  inline-size: 100%;
  block-size: 100%;
}
</style>
