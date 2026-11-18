<script setup lang="ts">
/**
 * The button at the end of the filter field that opens the searchable list of the files the results come from, each checked while
 * its results are shown; choosing a file shows or hides its results.
 */
import { computed, ref, useTemplateRef } from 'vue'
import {
  Button,
  SearchableListPopup,
  Tooltip,
  measureRect,
  useTooltip,
  type ListItem,
} from '@numen/ui'
import { WORDS as words } from '../../words'
import type { SourceView } from '../lib/view'
import { SourcesIcon } from '../model/icons'

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
const tip = useTooltip()
const anchor = ref<{ x: number; y: number; width: number } | null>(null)

const items = computed<readonly ListItem[]>(() =>
  props.sources.map((source) => ({ id: source.path, text: source.title, detail: source.path })),
)
const shown = computed(() =>
  props.sources.map((source) => source.path).filter((path) => !props.hidden.has(path)),
)
const isFiltered = computed(() => props.hidden.size > 0)
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
    variant="ghost"
    size="icon-small"
    class="saved-search__sources"
    :class="{ 'saved-search__sources--filtered': isFiltered }"
    :aria-label="words.sources"
    :aria-describedby="tip.describedBy.value"
    aria-haspopup="listbox"
    :aria-expanded="anchor !== null"
    @mouseenter="tip.open"
    @mouseleave="tip.close"
    @focus="tip.open"
    @blur="tip.close"
    @click="onOpen"
  >
    <SourcesIcon :size="16" aria-hidden="true" />
  </Button>
  <Teleport to="body">
    <Tooltip v-if="tip.at.value && anchor === null" :id="tip.id" :at="tip.at.value">
      {{ words.sources }}
    </Tooltip>
  </Teleport>
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
