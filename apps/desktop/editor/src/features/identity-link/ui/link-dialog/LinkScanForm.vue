<script setup lang="ts">
/**
 * Takes the text of the QR code of the other device: from the camera where the machine has one, and always from a
 * field to paste it into. It says when the camera is being checked, when it is ready and why it is not used.
 */
import { hasCamera, QrScanner, type ScannerCamera } from '@numen/ui'
import { useLinkScan } from '../../model/useLinkScan'
import { WORDS as words } from '../../words'
import LinkField from './LinkField.vue'
import LinkStatus from './LinkStatus.vue'
import LinkSteps from './LinkSteps.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    modelValue: string
    problem: string
    steps: readonly string[]
    findCamera?: () => Promise<boolean>
    camera?: ScannerCamera
  }>(),
  { findCamera: hasCamera },
)

const emit = defineEmits<{
  (e: 'update:modelValue', text: string): void
  (e: 'scanned', text: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const { phase, notice, setUnavailable } = useLinkScan(props.findCamera)
</script>

<template>
  <div class="link-scan">
    <LinkSteps :steps="steps" />
    <LinkStatus
      v-if="phase === 'checking'"
      :title="words.linkCheckingCamera"
      :detail="words.linkCheckingCameraDetail"
    />
    <div v-else-if="phase === 'camera'" class="link-scan__camera">
      <QrScanner
        v-bind="camera === undefined ? {} : { camera }"
        @scanned="emit('scanned', $event)"
        @unavailable="setUnavailable"
      />
      <p class="link-scan__point">{{ words.linkPointCamera }}</p>
    </div>
    <p v-else role="status" class="link-scan__notice">{{ notice }}</p>
    <LinkField
      :model-value="modelValue"
      :label="words.linkPaste"
      is-mono
      @update:model-value="emit('update:modelValue', $event)"
    />
    <p class="link-scan__problem" :role="problem === '' ? undefined : 'alert'">{{ problem }}</p>
  </div>
</template>

<style scoped>
.link-scan,
.link-scan__camera {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.link-scan__point {
  margin: 0;
  color: var(--numen-hushed);
  text-align: center;
}

.link-scan__notice {
  margin: 0;
  color: var(--numen-hushed);
}

.link-scan__problem {
  min-block-size: calc(1em * var(--numen-line-height));
  margin: 0;
  color: var(--numen-alarm);
}
</style>
