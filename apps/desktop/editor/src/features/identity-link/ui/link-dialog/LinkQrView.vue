<script setup lang="ts">
/** The QR code of this device, its text to copy where the other device has no camera, and what is waited for. */
import { Button, QrCode, Spinner } from '@numen/ui'
import { useCopyText } from '../../model/useCopyText'
import { WORDS as words } from '../../words'
import LinkField from './LinkField.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ qrText: string; accountFingerprint: string }>()

/* --------------------------------- State ---------------------------------- */
const QR_SIZE = 180
const { isCopied, copy } = useCopyText(() => props.qrText)
</script>

<template>
  <div class="link-qr">
    <div class="link-qr__code"><QrCode :text="qrText" :label="words.linkQr" :size="QR_SIZE" /></div>
    <LinkField :model-value="qrText" :label="words.linkText" is-readonly is-mono>
      <Button type="button" variant="outline" size="small" @click="copy">
        {{ isCopied ? words.linkCopied : words.linkCopy }}
      </Button>
    </LinkField>
    <p v-if="accountFingerprint !== ''" class="link-qr__account">
      <span class="link-qr__label">{{ words.account }}</span>
      <span class="link-qr__value">{{ accountFingerprint }}</span>
    </p>
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
  gap: 0.5rem;
}

.link-qr__code {
  display: flex;
  justify-content: center;
}

.link-qr__account {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem 0.75rem;
  margin: 0;
}

.link-qr__label {
  color: var(--numen-hushed);
}

.link-qr__value {
  font-family: var(--numen-font-mono);
  overflow-wrap: anywhere;
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
