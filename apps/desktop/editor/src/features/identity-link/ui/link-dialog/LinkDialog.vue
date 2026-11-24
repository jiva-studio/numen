<script setup lang="ts">
/**
 * The dialog of a device that has the account: it shows its QR code at once for the new device to scan, with three
 * steps and a button that copies the link for a device without a camera. The new device then asks to join, and this
 * dialog shows its name and the confirmation code to type on it. Escape closes it, and closing it cancels the link.
 */
import { computed, ref } from 'vue'
import { Dialog } from '@numen/ui'
import { getLinkView } from '../../lib/phases'
import type { LinkCalls } from '../../lib/types'
import { useCopyText } from '../../model/useCopyText'
import { useLinkActions } from '../../model/useLinkActions'
import { useLinkSession } from '../../model/useLinkSession'
import { WORDS as words } from '../../words'
import LinkActions from './LinkActions.vue'
import LinkProgress from './LinkProgress.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ calls: LinkCalls; stallMs?: number }>()

const emit = defineEmits<{ (e: 'close'): void }>()

/* --------------------------------- State ---------------------------------- */
const session = useLinkSession(() => props.calls, 'host', props.stallMs)
const code = ref('')
const view = computed(() =>
  getLinkView({
    link: session.link.value,
    role: 'host',
    isScanning: false,
    isStalled: session.isStalled.value,
    hasProblem: session.problem.value !== '',
  }),
)
const actions = useLinkActions(session, { text: ref(''), code }, view)
const { isCopied, copy } = useCopyText(() => session.link.value?.qrText ?? '')
const extra = computed(() =>
  view.value === 'qr' || view.value === 'preparing'
    ? {
        label: isCopied.value ? words.linkCopied : words.linkCopy,
        isDisabled: (session.link.value?.qrText ?? '') === '',
      }
    : undefined,
)
</script>

<template>
  <Dialog is-form :label="words.linkHostTitle" size="md" :step="view" @submit="actions.submit" @cancel="emit('close')">
    <LinkProgress
      v-model:code="code"
      :view="view"
      :link="session.link.value"
      role="host"
      :problem="session.problem.value"
    />
    <template #actions>
      <LinkActions
        :view="view"
        :is-busy="session.isBusy.value"
        :has-input="false"
        :is-pasting="false"
        v-bind="extra === undefined ? {} : { extra }"
        @close="emit('close')"
        @decline="actions.decline"
        @extra="copy"
      />
    </template>
  </Dialog>
</template>
