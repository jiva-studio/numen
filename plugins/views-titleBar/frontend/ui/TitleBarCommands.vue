<script setup lang="ts">
/**
 * The buttons of the bar, and the menu button that holds the ones that do not fit.
 */
import { computed, onMounted, onUpdated, ref, useTemplateRef, type Component } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { Menu, measureRect, type MenuItem } from '@numen/ui'
import { findGroupStarts } from '../lib/groups'
import { splitCommands } from '../lib/overflow'
import { formatTooltip } from '../lib/title'
import type { TitleBarCommand } from '@numen/plugin-sdk'
import type { TitleBarWords } from '../lib/words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  commands: readonly TitleBarCommand[]
  /** The width the commands may take in pixels, or null while it is not measured. */
  room: number | null
  words: TitleBarWords
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run-command', id: string): void }>()

/* --------------------------------- State ---------------------------------- */
const row = useTemplateRef<HTMLElement>('row')
const more = useTemplateRef<HTMLElement>('more')
const buttonWidth = ref(0)
const isOpen = ref(false)
const anchor = ref({ x: 0, y: 0 })
const triggerWidth = ref(0)

const capacity = computed(() =>
  props.room === null || buttonWidth.value === 0
    ? Infinity
    : Math.max(Math.floor(props.room / buttonWidth.value), 0),
)
const lists = computed(() => splitCommands(props.commands, capacity.value))
const isMenuOpen = computed(() => isOpen.value && lists.value.overflow.length > 0)
const starts = computed(() => findGroupStarts(lists.value.visible))
const items = computed<readonly MenuItem[]>(() =>
  lists.value.overflow.map(({ id, title, group }) => ({
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
  anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0 }
  triggerWidth.value = box?.width ?? 0
  isOpen.value = true
}

function onPickCommand(id: string): void {
  isOpen.value = false
  emit('run-command', id)
}

/* -------------------------------- Helpers --------------------------------- */
function measureButton(): void {
  const first = row.value?.querySelector<HTMLElement>('[data-title-bar-command]')
  if (first && first.offsetWidth > 0 && first.offsetWidth !== buttonWidth.value) {
    buttonWidth.value = first.offsetWidth
  }
}

function getIcon(id: string) {
  return props.commands.find((command) => command.id === id)?.icon as Component | undefined
}
</script>

<template>
  <div ref="row" class="title-bar__commands">
    <button
      v-for="command in lists.visible"
      :key="command.id"
      type="button"
      class="title-bar__button"
      :data-title-bar-command="command.id"
      :data-group-start="starts.has(command.id) ? '' : undefined"
      tabindex="-1"
      :title="formatTooltip(command)"
      style="--wails-draggable: no-drag"
      @click="emit('run-command', command.id)"
    >
      <component :is="getIcon(command.id)" :size="16" />
    </button>
    <button
      v-if="lists.overflow.length > 0"
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
      :at="anchor"
      :asking="triggerWidth"
      :from="more"
      :is-open="isMenuOpen"
      :name="words.more"
      @choose="onPickCommand"
      @dismiss="isOpen = false"
    >
      <template #icon="{ id }">
        <component :is="getIcon(id)" :size="16" />
      </template>
    </Menu>
  </div>
</template>
