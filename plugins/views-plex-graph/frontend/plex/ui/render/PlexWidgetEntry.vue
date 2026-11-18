<script setup lang="ts">
/** One view a plugin places at a node, drawn so that the views it draws are waited for until they have loaded. */
import { provideSlotLoading, type SlotLoading } from '@numen/plugin-sdk'
import type { PlexWidget } from '../../../types'

const props = defineProps<{
  entry: PlexWidget
  loading: SlotLoading
  path: string
}>()

provideSlotLoading(props.loading, {
  key: props.entry.key,
  pluginId: props.entry.key.split('/')[0] ?? '',
})
</script>

<template>
  <component :is="entry.component" :path="path" />
</template>
