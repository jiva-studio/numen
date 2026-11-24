<script setup lang="ts">
/**
 * The QR code of this device beside the steps, the account and what is waited for, and under both the text of the
 * link to copy where the other device has no camera. Below 520px of width the two columns stack.
 */
import { Button, QrCode, Spinner } from '@numen/ui'
import type { Fact } from '../../lib/types'
import { useCopyText } from '../../model/useCopyText'
import { WORDS as words } from '../../words'
import LinkFacts from './LinkFacts.vue'
import LinkField from './LinkField.vue'
import LinkSteps from './LinkSteps.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  qrText: string
  accountFingerprint: string
  steps: readonly string[]
  facts: readonly Fact[]
}>()

/* --------------------------------- State ---------------------------------- */
const QR_SIZE = 180
const { isCopied, copy } = useCopyText(() => props.qrText)
</script>

<template>
  <div class="link-qr">
    <div class="link-qr__code"><QrCode :text="qrText" :label="words.linkQr" :size="QR_SIZE" /></div>
    <div class="link-qr__side">
      <LinkSteps :steps="steps" />
      <LinkFacts v-if="facts.length > 0" :facts="facts" />
      <p v-if="accountFingerprint !== ''" class="link-qr__account">
        <span class="link-qr__label">{{ words.account }}</span>
        <span class="link-qr__value" :title="accountFingerprint">{{ accountFingerprint }}</span>
      </p>
      <p class="link-qr__wait" role="status" aria-busy="true">
        <Spinner />
        {{ words.linkWaitingScanTitle }}
      </p>
    </div>
    <div class="link-qr__link">
      <LinkField :model-value="qrText" :label="words.linkText" is-readonly is-mono>
        <Button type="button" variant="outline" size="small" @click="copy">
          {{ isCopied ? words.linkCopied : words.linkCopy }}
        </Button>
      </LinkField>
    </div>
  </div>
</template>

<style scoped>
.link-qr {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.5rem 1rem;
  container-type: inline-size;
}

.link-qr__code {
  display: flex;
  align-items: flex-start;
  justify-content: center;
}

.link-qr__side {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  min-inline-size: 0;
}

.link-qr__link {
  grid-column: 1 / -1;
}

.link-qr__account {
  display: flex;
  flex-direction: column;
  min-inline-size: 0;
  margin: 0;
}

.link-qr__label {
  color: var(--numen-hushed);
}

.link-qr__value {
  overflow: hidden;
  font-family: var(--numen-font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.link-qr__wait {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  color: var(--numen-hushed);
}

@container (max-width: 32.5rem) {
  .link-qr {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
