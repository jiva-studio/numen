<script setup lang="ts">
/**
 * A button of the bar that opens a menu of slot entries above it: a submenu a plugin placed in a
 * slot, or the items of a command that lists them. The items of the commands of the menu are read as
 * it opens. A pick emits the command of its row and the id of its item.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, type Component } from 'vue'
import { ChevronUp } from '@lucide/vue'
import { Menu, measureRect, type MenuTrigger } from '@numen/ui'
import { WORDS } from '../lib/entries'
import { BAR, useBarSlotMenu } from '../model/useBarSlotMenu'
import type { SlotMenuButtonProps } from './SlotMenuButton.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<SlotMenuButtonProps>(), { isItemList: false })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'run', command: string, itemId?: string, values?: Readonly<Record<string, string>>): void
}>()

/* --------------------------------- State ---------------------------------- */
const button = useTemplateRef<HTMLElement>('button')
const trigger = ref<MenuTrigger>('pointer')
const anchor = ref({ x: 0, y: 0 })
const slotMenu = useBarSlotMenu({
  getListItems: () => props.listItems,
  getReportError: () => props.reportError,
  words: WORDS,
})
const rows = computed(() => {
  const items = slotMenu.menu.value?.items ?? []
  const [first] = items
  return props.isItemList && first?.children ? first.children : items
})
const isMenuOpen = computed(() => slotMenu.menu.value !== null && rows.value.length > 0)

/* --------------------------------- Hooks ---------------------------------- */
onBeforeUnmount(() => slotMenu.close())

/* -------------------------------- Handlers -------------------------------- */
function onToggle(event: MouseEvent): void {
  if (isMenuOpen.value) {
    onClose()
    return
  }
  const box = measureRect(button.value)
  anchor.value = { x: box?.left ?? 0, y: box?.top ?? 0 }
  trigger.value = event.detail === 0 ? 'keyboard' : 'pointer'
  slotMenu.open(() => ({ target: BAR, entries: props.entries }))
}

function onClose(): void {
  slotMenu.close()
}

function onChoose(id: string): void {
  const pick = slotMenu.pick(id)
  if (!pick) return
  if (pick.values) emit('run', pick.command, pick.itemId, pick.values)
  else if (pick.itemId === undefined) emit('run', pick.command)
  else emit('run', pick.command, pick.itemId)
}

/* -------------------------------- Helpers --------------------------------- */
function getIcon(id: string): Component | undefined {
  return slotMenu.getIcon(id) as Component | undefined
}
</script>

<template>
  <button
    ref="button"
    type="button"
    class="status-bar__entry"
    data-status-bar-entry
    aria-haspopup="menu"
    :aria-expanded="isMenuOpen"
    :aria-label="label"
    :title="tooltip ?? label"
    @click="onToggle"
  >
    <component :is="icon" v-if="icon" :size="14" aria-hidden="true" />
    <template v-if="!icon || !isItemList">{{ label }}</template>
    <ChevronUp :size="12" aria-hidden="true" />
  </button>
  <Menu
    :items="rows"
    :at="anchor"
    :from="button"
    :trigger="trigger"
    :is-open="isMenuOpen"
    :name="label"
    @choose="onChoose"
    @dismiss="onClose"
  >
    <template #icon="{ id }">
      <component :is="getIcon(id)" v-if="getIcon(id)" :size="14" />
    </template>
  </Menu>
</template>
