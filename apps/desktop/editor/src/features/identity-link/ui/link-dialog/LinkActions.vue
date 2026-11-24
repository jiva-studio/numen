<script setup lang="ts">
/**
 * The row of buttons of a link dialog for its view: one extra way (copy the link, paste it instead), a way out and
 * the one step forward.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import type { LinkView } from '../../lib/phases'
import { getPrimaryAction } from '../../lib/primaryAction'
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
const primary = computed(() => getPrimaryAction(props.view, props))
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
