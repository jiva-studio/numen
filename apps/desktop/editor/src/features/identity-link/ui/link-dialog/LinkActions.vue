<script setup lang="ts">
/** The row of buttons of the link dialog for its view: a way to switch, a way out and the one step forward. */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import type { LinkView } from '../../lib/phases'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  view: LinkView
  isBusy: boolean
  hasInput: boolean
  switchLabel: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'decline'): void
  (e: 'switch'): void
}>()

/* --------------------------------- State ---------------------------------- */
const primary = computed(() => {
  switch (props.view) {
    case 'scan':
      return { label: words.linkUse, isDisabled: props.isBusy || !props.hasInput }
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
  <Button v-if="switchLabel !== ''" type="button" variant="ghost" @click="emit('switch')">
    {{ switchLabel }}
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
