<script setup lang="ts">
/** What a tab holds. Dialogs opened inside it cover and are centred in the element the tab is drawn into. */
import { computed } from 'vue'
import { provideDialogHost } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ host: HTMLElement | undefined }>()

/* --------------------------------- State ---------------------------------- */
provideDialogHost(computed(() => props.host ?? null))
</script>

<template>
  <slot />
</template>
