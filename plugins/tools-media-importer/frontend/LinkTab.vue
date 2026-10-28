<script setup lang="ts">
/**
 * A link's tab: the address it holds, the copy of its video where one is kept, the text fetched
 * from it, and the work a person starts on it.
 */
import { onUnmounted } from 'vue'
import LinkPlayer from './LinkPlayer.vue'
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
const tab = useLink(props.links, props.state?.path ?? props.path ?? '')
const { url, paragraphs, source, embedSource, error } = tab
const { isCopyable, isTextDeleted, isWorking, isLoaded } = tab.status

/* --------------------------------- Hooks ---------------------------------- */
onUnmounted(() => tab.close())

/* -------------------------------- Handlers -------------------------------- */
function onFetchText() {
  void tab.actions.fetchText()
}

function onCopyVideo() {
  void tab.actions.copyVideo()
}

function onDeleteCopy() {
  void tab.actions.deleteCopy()
}
</script>

<template>
  <article class="link-tab">
    <header class="link-tab__header">
      <a v-if="url" class="link-tab__address" :href="url" target="_blank" rel="noreferrer">
        {{ url }}
      </a>
      <div v-if="isLoaded && !error" class="link-tab__actions">
        <button type="button" class="link-tab__action" :disabled="isWorking" @click="onFetchText">
          {{ words.fetchText }}
        </button>
        <button
          v-if="isCopyable && !source"
          type="button"
          class="link-tab__action"
          :disabled="isWorking"
          @click="onCopyVideo"
        >
          {{ words.copyVideo }}
        </button>
        <button v-if="source" type="button" class="link-tab__action" @click="onDeleteCopy">
          {{ words.deleteCopy }}
        </button>
      </div>
    </header>
    <p v-if="error" class="link-tab__error" role="alert">{{ error }}</p>
    <LinkPlayer v-if="source" :source="source" />
    <iframe
      v-else-if="embedSource"
      class="link-tab__embed"
      :src="embedSource"
      :title="url"
      allow="fullscreen; picture-in-picture; encrypted-media"
      sandbox="allow-scripts allow-same-origin allow-popups allow-presentation"
    />
    <p v-else-if="isWorking && isCopyable" class="link-tab__status">{{ words.copying }}</p>
    <section v-if="paragraphs.length > 0" class="link-tab__text">
      <p v-for="(paragraph, index) in paragraphs" :key="index">{{ paragraph }}</p>
    </section>
    <p v-else-if="isWorking" class="link-tab__status">{{ words.fetching }}</p>
    <p v-else-if="isTextDeleted" class="link-tab__status">{{ words.textDeleted }}</p>
    <p v-else-if="isLoaded && !error" class="link-tab__status">{{ words.noText }}</p>
  </article>
</template>
