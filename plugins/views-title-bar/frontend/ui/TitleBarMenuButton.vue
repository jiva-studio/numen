<script setup lang="ts">
/**
 * A button of the bar that opens a menu of slot entries: the menu of the commands that do not fit,
 * a submenu a plugin placed in a slot, or the items of a command that lists them. The items of the
 * commands of the menu are read as it opens. A pick emits the command of its row and its argument.
 */
import { computed, onBeforeUnmount, ref, shallowRef, useTemplateRef, type Component } from 'vue'
import { Menu, measureRect, type MenuTrigger } from '@numen/ui'
import {
  createSlotMenu,
  readCommandItemLists,
  type CommandItem,
  type CommandItemLists,
  type Disposable,
  type SlotItem,
} from '@numen/plugin-sdk'
import { TITLE_BAR_WORDS } from '../lib/words'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    entries: readonly SlotItem[]
    /** What the button and its menu are called. */
    label: string
    /** The icon of the button; the label is written where there is none. */
    icon?: unknown
    /** The tooltip; the label where none is given. */
    tooltip?: string
    /** Whether the menu is the items of the one command of the entries, rather than its row. */
    isItemList?: boolean
    /** The items a command lists. */
    listItems: (command: string) => Promise<readonly CommandItem[]>
  }>(),
  { isItemList: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run-command', command: string, argument?: string): void }>()

/* --------------------------------- State ---------------------------------- */
const button = useTemplateRef<HTMLElement>('button')
const isOpen = ref(false)
const trigger = ref<MenuTrigger>('pointer')
const anchor = ref({ x: 0, y: 0 })
const width = ref(0)
const itemLists = shallowRef<CommandItemLists>(new Map())
let reading: Disposable | null = null
const menu = computed(() => createSlotMenu(props.entries, itemLists.value, TITLE_BAR_WORDS))
const rows = computed(() => {
  const [first] = menu.value.items
  return props.isItemList && first?.children ? first.children : menu.value.items
})
const isMenuOpen = computed(() => isOpen.value && rows.value.length > 0)

/* --------------------------------- Hooks ---------------------------------- */
onBeforeUnmount(() => reading?.dispose())

/* -------------------------------- Handlers -------------------------------- */
function onToggle(event: MouseEvent): void {
  if (isOpen.value) {
    onClose()
    return
  }
  const box = measureRect(button.value)
  anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0 }
  width.value = box?.width ?? 0
  trigger.value = event.detail === 0 ? 'keyboard' : 'pointer'
  reading?.dispose()
  reading = readCommandItemLists(
    props.entries,
    props.listItems,
    (lists) => (itemLists.value = lists),
  )
  isOpen.value = true
}

function onClose(): void {
  reading?.dispose()
  reading = null
  isOpen.value = false
}

function onChoose(id: string): void {
  const pick = menu.value.getPick(id)
  onClose()
  if (!pick) return
  if (pick.argument === undefined) emit('run-command', pick.command)
  else emit('run-command', pick.command, pick.argument)
}

/* -------------------------------- Helpers --------------------------------- */
function getIcon(id: string): Component | undefined {
  return menu.value.getIcon(id) as Component | undefined
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
    <component :is="icon as Component" v-if="icon !== undefined" :size="16" />
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
