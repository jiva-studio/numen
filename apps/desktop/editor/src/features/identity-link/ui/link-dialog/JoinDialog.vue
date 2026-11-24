<script setup lang="ts">
/**
 * The dialog of a new device: the camera scans the QR code of a device that has the account, and a button reveals a
 * field to paste its link into for a machine without a camera. Then it shows the name of this device, asks for
 * the confirmation code the other device shows, and asks whether to join the account. Escape closes it, and closing
 * it cancels the link.
 */
import { computed, ref } from 'vue'
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

const emit = defineEmits<{ (e: 'close'): void }>()

/* --------------------------------- State ---------------------------------- */
const session = useLinkSession(() => props.calls, 'joiner', props.stallMs)
const scan = useLinkScan(props.findCamera)
const draft = { text: ref(''), code: ref('') }
const isPasteAsked = ref(false)
const isPasting = computed(() => isPasteAsked.value || scan.phase.value === 'paste')
const actions = useLinkActions(session, 'joiner', draft)
const { view } = actions
const extra = computed(() =>
  view.value === 'scan' && !isPasting.value
    ? { label: words.linkPasteInstead, isDisabled: false }
    : undefined,
)
const hasInput = computed(() =>
  view.value === 'scan' ? draft.text.value !== '' : draft.code.value.length === 6,
)
</script>

<template>
  <Dialog
    is-form
    :label="words.linkJoinTitle"
    :hint="view === 'scan' ? words.linkJoinHint : ''"
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
      :notice="scan.notice.value"
      :is-pasting="isPasting"
      v-bind="camera === undefined ? {} : { camera }"
      @scanned="actions.scan"
      @unavailable="scan.setUnavailable"
    />
    <LinkProgress
      v-else
      v-model:code="draft.code.value"
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
        v-bind="extra === undefined ? {} : { extra }"
        @close="emit('close')"
        @decline="actions.decline"
        @extra="isPasteAsked = true"
      />
    </template>
  </Dialog>
</template>
