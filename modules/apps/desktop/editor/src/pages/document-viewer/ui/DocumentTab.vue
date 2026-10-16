<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import type { DocumentTabState } from '../model/useDocumentTab'
import PluginDocumentTab from '@plugins/reader-pdf/frontend/DocumentTab.vue'
import DocumentFallback from './DocumentFallback.vue'
import { isPdfPluginActive } from '../model/pluginState'

const props = defineProps<{ state: DocumentTabState }>()
</script>

<template>
  <PluginDocumentTab v-if="isPdfPluginActive" :state="props.state" />
  <DocumentFallback v-else :path="props.state.path" />
</template>
