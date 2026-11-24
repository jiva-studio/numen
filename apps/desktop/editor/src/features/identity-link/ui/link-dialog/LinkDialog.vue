<script setup lang="ts">
/**
 * The dialog that links a device to the account by a QR code, with the steps written on it. A device that has the
 * account shows its QR code at once and asks about the new device that scans it; a new device opens its camera to
 * scan the QR code of the other device, or shows its own. The device that has the account then shows a confirmation
 * code, and the new device asks for it and for the answer to join the account. Escape closes it, and closing it
 * cancels the link.
 */
import { computed, ref } from 'vue'
import { Dialog, type ScannerCamera } from '@numen/ui'
import type { LinkRole } from '@/entities/identity'
import { getLinkView } from '../../lib/phases'
import type { LinkCalls } from '../../lib/types'
import { useLinkActions } from '../../model/useLinkActions'
import { useLinkSession } from '../../model/useLinkSession'
import { WORDS as words } from '../../words'
import LinkActions from './LinkActions.vue'
import LinkProgress from './LinkProgress.vue'
import LinkScanForm from './LinkScanForm.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  role: LinkRole
  calls: LinkCalls
  findCamera?: () => Promise<boolean>
  camera?: ScannerCamera
  stallMs?: number
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

/* --------------------------------- State ---------------------------------- */
const session = useLinkSession(() => props.calls, props.role, props.stallMs)
const draft = { text: ref(''), code: ref(''), isScanning: ref(props.role === 'joiner') }
const view = computed(() =>
  getLinkView({
    link: session.link.value,
    role: props.role,
    isScanning: draft.isScanning.value,
    isStalled: session.isStalled.value,
    hasProblem: session.problem.value !== '',
  }),
)
const actions = useLinkActions(session, draft, view)
const steps = computed(() =>
  words.linkSteps(props.role, view.value === 'scan' ? 'scan' : (session.link.value?.mode ?? 'show')),
)
const switchLabel = computed(() => {
  if (view.value === 'scan') return words.linkShowInstead
  return view.value === 'qr' ? words.linkScanInstead : ''
})
const isCompact = computed(() => view.value === 'qr' || view.value === 'preparing')
const hint = computed(() => {
  if (isCompact.value) return ''
  return props.role === 'host' ? words.linkHostHint : words.linkJoinHint
})
const hasInput = computed(() =>
  view.value === 'scan' ? draft.text.value !== '' : draft.code.value.length === 6,
)
</script>

<template>
  <Dialog
    is-form
    :label="role === 'host' ? words.linkHostTitle : words.linkJoinTitle"
    :hint="hint"
    :size="isCompact ? 'lg' : 'md'"
    :step="view"
    @submit="actions.submit"
    @cancel="emit('close')"
  >
    <LinkScanForm
      v-if="view === 'scan'"
      v-model="draft.text.value"
      :problem="session.problem.value"
      :steps="steps"
      v-bind="{
        ...(findCamera === undefined ? {} : { findCamera }),
        ...(camera === undefined ? {} : { camera }),
      }"
      @scanned="actions.scan"
    />
    <LinkProgress
      v-else
      v-model:code="draft.code.value"
      :view="view"
      :link="session.link.value"
      :role="role"
      :problem="session.problem.value"
      :steps="steps"
    />
    <template #actions>
      <LinkActions
        :view="view"
        :is-busy="session.isBusy.value"
        :has-input="hasInput"
        :switch-label="switchLabel"
        @close="emit('close')"
        @decline="actions.decline"
        @switch="actions.switchWay"
      />
    </template>
  </Dialog>
</template>
