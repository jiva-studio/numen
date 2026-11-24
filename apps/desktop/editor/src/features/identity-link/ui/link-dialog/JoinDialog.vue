<script setup lang="ts">
/**
 * The dialog of a new device: the camera scans the QR code of a device that has the account, and a machine
 * without a camera shows a field to paste its link into instead. Then it shows the name of this device and asks
 * whether to join the identity. Escape closes it, and closing
 * it cancels the link.
 */
import { computed, ref, watch } from 'vue'
import { Dialog, hasCamera, type ScannerCamera } from '@numen/ui'
import type { LinkCalls } from '../../lib/types'
import { useLinkActions } from '../../model/useLinkActions'
import { useLinkScan } from '../../model/useLinkScan'
import { useLinkSession } from '../../model/useLinkSession'
import { WORDS as words } from '../../words'
import LinkActions from './LinkActions.vue'
import LinkProgress from './LinkProgress.vue'
import LinkScanForm from './LinkScanForm.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    calls: LinkCalls
    findCamera?: () => Promise<boolean>
    camera?: ScannerCamera
    stallMs?: number
  }>(),
  { findCamera: hasCamera },
)

const emit = defineEmits<{ (e: 'close'): void; (e: 'linked'): void }>()

/* --------------------------------- State ---------------------------------- */
const session = useLinkSession(() => props.calls, 'joiner', props.stallMs)
const scan = useLinkScan(props.findCamera)
const draft = { text: ref('') }
const isPasting = computed(() => scan.phase.value === 'paste')
const actions = useLinkActions(session, 'joiner', draft)
const { view } = actions
watch(
  () => session.link.value?.phase,
  (phase) => {
    if (phase === 'linked') emit('linked')
  },
)
const hasInput = computed(() => view.value === 'scan' && draft.text.value !== '')

/* -------------------------------- Handlers -------------------------------- */
function getHint(): string {
  if (view.value !== 'scan') return ''
  return isPasting.value ? words.linkJoinPasteHint : words.linkJoinHint
}
</script>

<template>
  <Dialog
    is-form
    :label="words.linkJoinTitle"
    :hint="getHint()"
    size="md"
    :step="view"
    @submit="actions.submit"
    @cancel="emit('close')"
  >
    <LinkScanForm
      v-if="view === 'scan'"
      v-model="draft.text.value"
      :problem="session.problem.value"
      :phase="scan.phase.value"
      v-bind="camera === undefined ? {} : { camera }"
      @scanned="actions.scan"
      @unavailable="scan.setUnavailable"
    />
    <LinkProgress
      v-else
      :view="view"
      :link="session.link.value"
      role="joiner"
      :problem="session.problem.value"
    />
    <template #actions>
      <LinkActions
        :view="view"
        :is-busy="session.isBusy.value"
        :has-input="hasInput"
        :is-pasting="isPasting"
        @close="emit('close')"
        @decline="actions.decline"
      />
    </template>
  </Dialog>
</template>
