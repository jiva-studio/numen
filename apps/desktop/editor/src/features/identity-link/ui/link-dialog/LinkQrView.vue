<script setup lang="ts">
/**
 * The QR code of this device above the steps and what is waited for. The box of the QR code keeps its size from the
 * first frame and holds a spinner until the text of the code arrives, so nothing moves when it does.
 */
import { QrCode, Spinner } from '@numen/ui'
import { WORDS as words } from '../../words'
import LinkSteps from './LinkSteps.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ qrText: string }>()

/* --------------------------------- State ---------------------------------- */
const QR_SIZE = 200
</script>

<template>
  <div class="link-qr">
    <div class="link-qr__code" :style="{ '--qr-size': `${QR_SIZE}px` }">
      <QrCode v-if="qrText !== ''" :text="qrText" :label="words.linkQr" :size="QR_SIZE" />
      <Spinner v-else />
    </div>
    <LinkSteps :steps="words.linkHostSteps" :is-waiting="qrText !== ''" />
    <p class="link-qr__wait" role="status" aria-busy="true">
      <Spinner />
      {{ qrText === '' ? words.linkPreparingTitle : words.linkWaitingScanTitle }}
    </p>
  </div>
</template>

<style scoped>
.link-qr {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.link-qr__code {
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

.link-qr__wait {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  color: var(--numen-hushed);
}
</style>
