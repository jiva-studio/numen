<script setup lang="ts">
/**
 * A link's tab: the address it holds, the copy of its video where one is kept, and the text
 * fetched from it.
 */
import { onUnmounted } from 'vue'
import LinkPlayer from './LinkPlayer.vue'
import { links as servedLinks } from './api/wire'
import { useLink } from './model/link'
import type { Links } from './types'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly path?: string
  readonly state?: { readonly path?: string }
  /** What the tab reads the link through. The window's own by default. */
  readonly links?: Links
}>()

/* --------------------------------- State ---------------------------------- */
const tab = useLink(props.links ?? servedLinks, props.state?.path ?? props.path ?? '')
const { url, paragraphs, source, isCopyable, isWorking, isLoaded, error } = tab

/* --------------------------------- Hooks ---------------------------------- */
onUnmounted(() => tab.close())
</script>

<template>
  <article class="link-tab">
    <header class="link-tab__header">
      <a v-if="url" class="link-tab__address" :href="url" target="_blank" rel="noreferrer">
        {{ url }}
      </a>
    </header>
    <p v-if="error" class="link-tab__error" role="alert">{{ error }}</p>
    <LinkPlayer v-if="source" :source="source" />
    <p v-else-if="isWorking && isCopyable" class="link-tab__status">{{ words.copying }}</p>
    <section v-if="paragraphs.length > 0" class="link-tab__text">
      <p v-for="(paragraph, index) in paragraphs" :key="index">{{ paragraph }}</p>
    </section>
    <p v-else-if="isWorking" class="link-tab__status">{{ words.fetching }}</p>
    <p v-else-if="isLoaded && !error" class="link-tab__status">{{ words.noText }}</p>
  </article>
</template>
