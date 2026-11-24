<script setup lang="ts">
/**
 * The row of buttons of a link dialog for its view: one extra way (copy the link, paste it instead), a way out and
 * the one step forward.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import type { LinkView } from '../../lib/phases'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  view: LinkView
  isBusy: boolean
  hasInput: boolean
  isPasting: boolean
  extra?: { label: string; isDisabled: boolean }
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'decline'): void
  (e: 'extra'): void
}>()

/* --------------------------------- State ---------------------------------- */
const primary = computed(() => {
  switch (props.view) {
    case 'scan':
      return props.isPasting
        ? { label: words.linkUse, isDisabled: props.isBusy || !props.hasInput }
        : undefined
    case 'peer':
      return { label: words.linkApprove, isDisabled: props.isBusy }
    case 'code':
      return { label: words.linkCodeConfirm, isDisabled: props.isBusy || !props.hasInput }
    case 'account':
      return { label: words.linkJoin, isDisabled: props.isBusy }
    case 'stalled':
    case 'refused':
    case 'failed':
      return { label: words.linkTryAgain, isDisabled: props.isBusy }
    default:
      return undefined
  }
})
const isAsking = computed(() => props.view === 'peer' || props.view === 'account')
const isDone = computed(() => props.view === 'linked')
</script>

<template>
  <Button
    v-if="extra"
    type="button"
    variant="outline"
    :disabled="extra.isDisabled"
    @click="emit('extra')"
  >
    {{ extra.label }}
  </Button>
  <Button
    v-if="!isDone"
    type="button"
    variant="ghost"
    @click="isAsking ? emit('decline') : emit('close')"
  >
    {{ view === 'peer' ? words.linkDecline : isAsking ? words.linkCancel : words.linkClose }}
  </Button>
  <Button v-if="primary" type="submit" variant="solid" :disabled="primary.isDisabled">
    {{ primary.label }}
  </Button>
  <Button v-if="isDone" type="button" variant="solid" @click="emit('close')">
    {{ words.linkDone }}
  </Button>
</template>
