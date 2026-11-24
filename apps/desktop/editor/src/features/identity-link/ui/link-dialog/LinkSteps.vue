<script setup lang="ts">
/** The steps a person follows on the two devices, in one fixed list: the step being waited for is the active one. */
import { computed } from 'vue'
import { StageList, type Stage } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ steps: readonly string[]; isWaiting: boolean }>()

/* --------------------------------- State ---------------------------------- */
const stages = computed<Stage[]>(() =>
  props.steps.map((label, at) => ({
    id: `step-${at}`,
    label,
    state: props.isWaiting && at === 0 ? 'active' : 'pending',
  })),
)
</script>

<template>
  <StageList :stages="stages" />
</template>
