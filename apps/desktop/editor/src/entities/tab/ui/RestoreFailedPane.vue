<script setup lang="ts">
/**
 * Placeholder displayed in place of a saved tab that could not be restored. The other saved tabs
 * open as usual.
 */
import { ErrorState } from '@numen/ui'
import type { RestoreFailedState } from '../model/restoreFailed'

defineProps<{
  state: RestoreFailedState
}>()
</script>

<template>
  <ErrorState heading="Could not restore this tab" :message="state.message" />
</template>
