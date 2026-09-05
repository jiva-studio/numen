<script setup lang="ts">
/**
 * A document tab: the pages of one document, and what is highlighted on them.
 *
 * What the document could not be read as is said where the pages would be.
 */
import { Reader } from '@numen/ui'
import { WORDS as words } from './words'
import type { DocumentTabState } from './kind'

const props = defineProps<{ held: DocumentTabState }>()
</script>

<template>
  <Reader
    :ref="(reader: unknown) => props.held.drew(reader)"
    :pages="props.held.pages.value"
    :sheets="props.held.sheets.value"
    :at="props.held.at.value"
    :picture="props.held.pictureOf"
    :highlights="props.held.highlightedOn"
    :also="props.held.alsoOn"
    :words="words"
    @go="(page: number) => void props.held.go(page)"
    @wide="(wide: number) => props.held.widen(wide)"
  >
    <template #silence>{{ props.held.trouble.value }}</template>
  </Reader>
</template>
