<script setup lang="ts">
/** The QR code of this device and its text, to copy where the other device has no camera. */
import { QrCode } from '@numen/ui'
import { WORDS as words } from '../../words'
import LinkFacts from './LinkFacts.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ qrText: string; accountFingerprint: string }>()
</script>

<template>
  <div class="link-qr">
    <QrCode :text="qrText" :label="words.linkQr" />
    <label class="link-qr__field">
      <span>{{ words.linkText }}</span>
      <input type="text" readonly :value="qrText" />
    </label>
    <LinkFacts
      v-if="accountFingerprint !== ''"
      :facts="[{ label: words.account, value: accountFingerprint, isMono: true }]"
    />
  </div>
</template>

<style scoped>
.link-qr,
.link-qr__field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
</style>
