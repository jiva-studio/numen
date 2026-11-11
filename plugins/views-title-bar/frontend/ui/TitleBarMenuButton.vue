<script setup lang="ts">
/**
 * A button of the bar that opens a menu of slot entries: the menu of the commands that do not fit,
 * a submenu a plugin placed in a slot, or the items of a command that lists them. The items of the
 * commands of the menu are read as it opens. A pick emits the command of its row and the id of its item.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, type Component } from 'vue'
import { Menu, measureRect, type MenuTrigger } from '@numen/ui'
import { TITLE_BAR_WORDS } from '../lib/words'
import { BAR, useBarSlotMenu } from '../model/useBarSlotMenu'
import type { TitleBarMenuButtonProps } from './TitleBarMenuButton.types'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<TitleBarMenuButtonProps>(), { isItemList: false })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (
    event: 'run-command',
    command: string,
    itemId?: string,
    values?: Readonly<Record<string, string>>,
  ): void
}>()

/* --------------------------------- State ---------------------------------- */
const button = useTemplateRef<HTMLElement>('button')
const trigger = ref<MenuTrigger>('pointer')
const anchor = ref({ x: 0, y: 0 })
const width = ref(0)
const slotMenu = useBarSlotMenu({
  getListItems: () => props.listItems,
  getReportError: () => props.reportError,
  words: TITLE_BAR_WORDS,
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
  anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0 }
  width.value = box?.width ?? 0
  trigger.value = event.detail === 0 ? 'keyboard' : 'pointer'
  slotMenu.open(() => ({ target: BAR, entries: props.entries }))
}

function onClose(): void {
  slotMenu.close()
}

function onChoose(id: string): void {
  const pick = slotMenu.pick(id)
  if (!pick) return
  if (pick.values) emit('run-command', pick.command, pick.itemId, pick.values)
  else if (pick.itemId === undefined) emit('run-command', pick.command)
  else emit('run-command', pick.command, pick.itemId)
}

/* -------------------------------- Helpers --------------------------------- */
function getIcon(id: string): Component | undefined {
  return slotMenu.getIcon(id) as Component | undefined
}
</script>

<template>
  <button
    v-bind="$attrs"
    ref="button"
    type="button"
    class="title-bar__button"
    :class="{ 'title-bar__button--text': icon === undefined }"
    data-title-bar-focusable
    aria-haspopup="menu"
    :aria-expanded="isMenuOpen"
    :aria-label="label"
    :title="tooltip ?? label"
    style="--wails-draggable: no-drag"
    @click="onToggle"
  >
    <component :is="icon" v-if="icon !== undefined" :size="16" />
    <template v-else>{{ label }}</template>
  </button>
  <Menu
    :items="rows"
    :at="anchor"
    :asking="width"
    :from="button"
    :trigger="trigger"
    :is-open="isMenuOpen"
    :name="label"
    @choose="onChoose"
    @dismiss="onClose"
  >
    <template #icon="{ id }">
      <component :is="getIcon(id)" v-if="getIcon(id)" :size="16" />
    </template>
  </Menu>
</template>
