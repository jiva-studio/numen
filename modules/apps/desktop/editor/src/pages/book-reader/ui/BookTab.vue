<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import type { BookTabState } from '../model/useBookTab'
import PluginBookTab from '@plugins/reader-epub/frontend/BookTab.vue'
import BookFallback from './BookFallback.vue'
import { isBookPluginActive } from '../model/pluginState'

const props = defineProps<{ state: BookTabState }>()
</script>

<template>
  <PluginBookTab v-if="isBookPluginActive" :state="props.state" />
  <BookFallback v-else :path="props.state.path" />
</template>
