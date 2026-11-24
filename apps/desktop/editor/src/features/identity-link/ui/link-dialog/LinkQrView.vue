<script setup lang="ts">
/** The QR code of this device, its text to copy where the other device has no camera, and what is waited for. */
import { QrCode, Spinner } from '@numen/ui'
import { WORDS as words } from '../../words'
import LinkField from './LinkField.vue'
import LinkFacts from './LinkFacts.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ qrText: string; accountFingerprint: string }>()
</script>

<template>
  <div class="link-qr">
    <div class="link-qr__code"><QrCode :text="qrText" :label="words.linkQr" /></div>
    <LinkField :model-value="qrText" :label="words.linkText" is-readonly is-mono />
    <LinkFacts
      v-if="accountFingerprint !== ''"
      :facts="[{ label: words.account, value: accountFingerprint, isMono: true }]"
    />
    <p class="link-qr__wait" role="status" aria-busy="true">
      <Spinner />
      {{ words.linkWaitingScanTitle }}
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
  justify-content: center;
}

.link-qr__wait {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  margin: 0;
  color: var(--numen-hushed);
}
</style>
