<script setup lang="ts">
/**
 * A button of the bar that opens a menu of slot entries: the menu of the commands that do not fit,
 * a submenu a plugin placed in a slot, or the items of a command that lists them. The items of the
 * commands of the menu are read as it opens. A pick emits the command of its row and its argument.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, type Component } from 'vue'
import { Menu, measureRect, type MenuTrigger } from '@numen/ui'
import { useSlotMenu, type CommandItem, type SlotItem } from '@numen/plugin-sdk'
import { TITLE_BAR_WORDS } from '../lib/words'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    entries: readonly SlotItem[]
    label: string
    icon?: unknown
    tooltip?: string
    isItemList?: boolean
    listItems: (command: string) => Promise<readonly CommandItem[]>
    reportError?: (error: unknown) => void
  }>(),
  { isItemList: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run-command', command: string, argument?: string): void }>()

/* --------------------------------- State ---------------------------------- */
const button = useTemplateRef<HTMLElement>('button')
const trigger = ref<MenuTrigger>('pointer')
const anchor = ref({ x: 0, y: 0 })
const width = ref(0)
/** The bar passes no target of its own; the window runs a bar entry over the front tab. */
const BAR = 'bar'
const slotMenu = useSlotMenu<typeof BAR>({
  listItems: (command) => props.listItems(command),
  words: TITLE_BAR_WORDS,
  onError: (error) => props.reportError?.(error),
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
  if (pick.argument === undefined) emit('run-command', pick.command)
  else emit('run-command', pick.command, pick.argument)
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
