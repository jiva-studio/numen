<script setup lang="ts">
/** One open dialog of a plugin: it draws the component of the plugin and holds an error the component throws, so the other dialogs and the window go on. */
import { onErrorCaptured } from 'vue'
import type { OpenDialog } from '../model/createDialogStack'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ entry: OpenDialog }>()

const emit = defineEmits<{ (e: 'fail'): void }>()

/* ---------------------------------- Hooks --------------------------------- */
onErrorCaptured(() => {
  emit('fail')
  return false
})
</script>

<template>
  <component :is="entry.options.component" v-bind="entry.options.props ?? {}" />
</template>
