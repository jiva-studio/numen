<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import type { PlexTabState } from '../types'
import PluginPlexTab from '@plugins/view-plex-graph/frontend/PlexGraphView.vue'
import PlexFallback from './PlexFallback.vue'
import { isPlexPluginActive } from '../model/pluginState'

const props = defineProps<{ state: PlexTabState }>()
</script>

<template>
  <PluginPlexTab v-if="isPlexPluginActive" :state="props.state" />
  <PlexFallback v-else />
</template>
