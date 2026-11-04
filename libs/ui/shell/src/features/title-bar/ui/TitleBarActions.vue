<script setup lang="ts">
/**
 * The buttons of the bar, and the menu button that holds the ones that do not fit.
 */
import { computed, onMounted, onUpdated, ref, useTemplateRef } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { Menu, measureRect, type MenuItem } from '@numen/ui'
import { findGroupStarts } from '../lib/groups'
import { splitActions } from '../lib/overflow'
import { formatTooltip } from '../lib/title'
import type { TitleBarAction } from '../lib/types'
import type { TitleBarWords } from '../lib/words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  actions: readonly TitleBarAction[]
  /** The width the actions may take in pixels, or null while it is not measured. */
  room: number | null
  words: TitleBarWords
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run', id: string): void }>()

/* --------------------------------- State ---------------------------------- */
const row = useTemplateRef<HTMLElement>('row')
const more = useTemplateRef<HTMLElement>('more')
const buttonWidth = ref(0)
const isOpen = ref(false)
const at = ref({ x: 0, y: 0 })
const asking = ref(0)

const capacity = computed(() =>
  props.room === null || buttonWidth.value === 0
    ? Infinity
    : Math.max(Math.floor(props.room / buttonWidth.value), 0),
)
const split = computed(() => splitActions(props.actions, capacity.value))
const starts = computed(() => findGroupStarts(split.value.visible))
const items = computed<readonly MenuItem[]>(() =>
  split.value.overflow.map(({ id, title, group }) => ({
    id,
    text: title,
    ...(group === undefined ? {} : { group }),
  })),
)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(measureButton)
onUpdated(measureButton)

/* -------------------------------- Handlers -------------------------------- */
function onOpenMenu(): void {
  const box = measureRect(more.value)
  at.value = { x: box?.left ?? 0, y: box?.bottom ?? 0 }
  asking.value = box?.width ?? 0
  isOpen.value = true
}

function onChoose(id: string): void {
  isOpen.value = false
  emit('run', id)
}

/* -------------------------------- Helpers --------------------------------- */
function measureButton(): void {
  const first = row.value?.querySelector<HTMLElement>('[data-title-bar-action]')
  if (first && first.offsetWidth > 0 && first.offsetWidth !== buttonWidth.value) {
    buttonWidth.value = first.offsetWidth
  }
}

function getIcon(id: string) {
  return props.actions.find((action) => action.id === id)?.icon
}
</script>

<template>
  <div ref="row" class="title-bar__actions">
    <button
      v-for="action in split.visible"
      :key="action.id"
      type="button"
      class="title-bar__button"
      :data-title-bar-action="action.id"
      :data-group-start="starts.has(action.id) ? '' : undefined"
      tabindex="-1"
      :title="formatTooltip(action)"
      style="--wails-draggable: no-drag"
      @click="emit('run', action.id)"
    >
      <component :is="action.icon" :size="16" />
    </button>
    <button
      v-if="split.overflow.length > 0"
      ref="more"
      type="button"
      class="title-bar__button"
      data-title-bar-more
      tabindex="-1"
      :title="words.more"
      style="--wails-draggable: no-drag"
      @click="onOpenMenu"
    >
      <Ellipsis :size="16" />
    </button>
    <Menu
      :items="items"
      :at="at"
      :asking="asking"
      :from="more"
      :is-open="isOpen"
      :name="words.more"
      @choose="onChoose"
      @dismiss="isOpen = false"
    >
      <template #icon="{ id }">
        <component :is="getIcon(id)" :size="16" />
      </template>
    </Menu>
  </div>
</template>
