<script setup lang="ts">
/** The three steps of linking a device, in one fixed list through every phase: done ones checked, the awaited one active. */
import { computed } from 'vue'
import { StageList, type Stage } from '@numen/ui'
import type { LinkView } from '../../lib/phases'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ view: LinkView }>()

/* --------------------------------- State ---------------------------------- */
const ACTIVE_STEP: Partial<Record<LinkView, number>> = { qr: 0, peer: 2, code: 2, linking: 2 }

const stages = computed<Stage[]>(() => {
  const active = ACTIVE_STEP[props.view] ?? (props.view === 'linked' ? words.linkHostSteps.length : -1)
  return words.linkHostSteps.map((label, at) => ({
    id: `step-${at}`,
    label,
    state: at < active ? 'done' : at === active ? 'active' : 'pending',
  }))
})
</script>

<template>
  <StageList :stages="stages" />
</template>
