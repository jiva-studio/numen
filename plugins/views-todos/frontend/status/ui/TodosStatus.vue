<script setup lang="ts">
/**
 * The chip of the title bar and the popover with the lists Overdue, Today and No date.
 */
import type { Summary, TaskReader } from '../../types'

defineProps<{
  summary: Summary | null
  api: TaskReader
  getToday: () => string
}>()

defineEmits<{
  (event: 'open', target: { path: string; line: number }): void
}>()
</script>

<template>
  <div />
</template>
