<script setup lang="ts">
/**
 * The button that opens the searchable list of the files the results come from, each checked while
 * its results are shown; choosing a file shows or hides its results.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { Button, SearchableListPopup, measureRect, type ListItem } from '@numen/ui'
import { WORDS as words } from '../../words'
import type { SourceView } from '../lib/view'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  sources: readonly SourceView[]
  hidden: ReadonlySet<string>
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'toggle', path: string, isShown: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof Button>>('trigger')
const anchor = ref<{ x: number; y: number; width: number } | null>(null)

const items = computed<readonly ListItem[]>(() =>
  props.sources.map((source) => ({ id: source.path, text: source.title, detail: source.path })),
)
const shown = computed(() =>
  props.sources.map((source) => source.path).filter((path) => !props.hidden.has(path)),
)
const triggerElement = computed(() => (trigger.value?.$el as HTMLElement | undefined) ?? null)

/* -------------------------------- Handlers -------------------------------- */
function onOpen() {
  const box = measureRect(triggerElement.value)
  anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0, width: box?.width ?? 0 }
}

function onClose() {
  anchor.value = null
}

function onSelect(path: string) {
  emit('toggle', path, props.hidden.has(path))
}
</script>

<template>
  <Button
    ref="trigger"
    variant="outline"
    aria-haspopup="listbox"
    :aria-expanded="anchor !== null"
    @click="onOpen"
  >
    {{ words.sources }}
  </Button>
  <SearchableListPopup
    v-if="anchor"
    :items="items"
    :chosen="shown"
    :label="words.sources"
    :placeholder="words.searchSources"
    :empty-text="words.noSourceMatch"
    :at="{ x: anchor.x, y: anchor.y }"
    :asking="anchor.width"
    :from="triggerElement"
    @select="onSelect"
    @close="onClose"
  />
</template>
