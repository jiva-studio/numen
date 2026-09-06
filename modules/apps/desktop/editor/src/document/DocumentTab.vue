<script setup lang="ts">
/**
 * A document tab: the pages of one document, and what is highlighted on them.
 *
 * What the document could not be read as is said where the pages would be.
 */
import { PageReader } from '@numen/ui'
import { WORDS as words } from './words'
import type { DocumentTabState } from './kind'

const props = defineProps<{ state: DocumentTabState }>()
</script>

<template>
  <PageReader
    :ref="(reader: unknown) => props.state.drew(reader)"
    :pages="props.state.pages.value"
    :sheets="props.state.sheets.value"
    :at="props.state.at.value"
    :picture="props.state.pictureOf"
    :highlights="props.state.highlightedOn"
    :also="props.state.alsoOn"
    :words="words"
    @go="(page: number) => void props.state.go(page)"
    @wide="(wide: number) => props.state.widen(wide)"
  >
    <template #silence>{{ props.state.trouble.value }}</template>
  </PageReader>
</template>
