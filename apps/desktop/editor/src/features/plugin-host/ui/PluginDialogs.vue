<script setup lang="ts">
/** The dialogs the plugins have open, one over another, each drawn by the component its plugin gave. */
import type { OpenDialog } from '../model/createDialogStack'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ entries: readonly OpenDialog[] }>()
</script>

<template>
  <component
    :is="one.options.component"
    v-for="one in entries"
    :key="one.id"
    v-bind="one.options.props ?? {}"
  />
</template>
