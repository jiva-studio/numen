<script setup lang="ts">
/**
 * A note: its Markdown set as prose, with the wikilinks as links. A press on the prose opens the text
 * for writing, and a press outside it sets the prose again.
 */
import { computed, nextTick, ref, useTemplateRef } from 'vue'
import { Prose, type LinkPiece } from '@numen/ui'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ body: string }>()

const emit = defineEmits<{
  'update:body': [body: string]
  open: [address: string]
}>()

/* --------------------------------- State ---------------------------------- */
const WIKILINK = /\[\[([^\]]+)\]\]/g
const LINK_PREFIX = 'wikilink:'

const field = useTemplateRef<HTMLTextAreaElement>('field')
const isWriting = ref(false)

const links = computed<LinkPiece[]>(() =>
  [...new Set(props.body.match(WIKILINK) ?? [])].map((raw) => {
    const name = raw.slice(2, -2)
    return { raw, href: `${LINK_PREFIX}${name}`, text: name }
  }),
)

/* -------------------------------- Handlers -------------------------------- */
async function onStartWriting() {
  isWriting.value = true
  await nextTick()
  field.value?.focus()
}

function onFollow(href: string) {
  if (href.startsWith(LINK_PREFIX)) emit('open', href.slice(LINK_PREFIX.length))
}

function onInput(event: Event) {
  emit('update:body', (event.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <div class="note">
    <textarea
      v-if="isWriting"
      ref="field"
      class="note__field"
      :value="body"
      spellcheck="false"
      aria-label="Note"
      @input="onInput"
      @blur="isWriting = false"
    />
    <div v-else class="note__prose" @click="onStartWriting">
      <Prose :text="body" :links="links" @follow="onFollow" />
    </div>
  </div>
</template>

<style scoped>
.note {
  block-size: 100%;
  overflow-y: auto;
  padding: 0.75rem var(--numen-gutter);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-reading-size);
  color: var(--numen-ink);
  user-select: text;
  -webkit-user-select: text;
}

.note__prose {
  min-block-size: 100%;
  cursor: text;
}

.note__field {
  display: block;
  inline-size: 100%;
  min-block-size: 100%;
  padding: 0;
  border: 0;
  outline: 0;
  resize: none;
  background: transparent;
  color: inherit;
  font: inherit;
  line-height: var(--numen-line-height);
  caret-color: var(--numen-caret);
}
</style>
