<script setup lang="ts">
/**
 * Document tab view rendering reader pages and highlights.
 */
import { Reader } from '@numen/ui'
import { WORDS as words } from './words'
import type { DocumentTabState } from './kind'

// --- Props & Emits ---
const props = defineProps<{ state: DocumentTabState }>()

// --- State ---

// --- Handlers ---
function onGoToPage(page: number) {
  void props.state.go(page)
}

function onWiden(wide: number) {
  props.state.widen(wide)
}

// --- Helpers ---
</script>

<template>
  <Reader
    :ref="(reader: unknown) => props.state.setPageHandle(reader)"
    :pages="props.state.pages.value"
    :at="props.state.at.value"
    :picture="props.state.pictureOf"
    :highlights="props.state.highlightedOn"
    :also="props.state.alsoOn"
    :words="words"
    @go="onGoToPage"
    @wide="onWiden"
  >
    <template #silence>{{ props.state.trouble.value }}</template>
  </Reader>
</template>
