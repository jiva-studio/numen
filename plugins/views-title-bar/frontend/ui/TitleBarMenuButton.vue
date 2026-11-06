<script setup lang="ts">
/**
 * A button of the bar that opens a menu of slot entries: the menu of the commands that do not fit,
 * or a submenu a plugin placed in a slot. A pick emits the command of its row.
 */
import { computed, ref, useTemplateRef, type Component } from 'vue'
import { Menu, measureRect, type MenuTrigger } from '@numen/ui'
import type { SlotItem } from '@numen/plugin-sdk'
import { createBarMenu } from '../lib/menu'

defineOptions({ inheritAttrs: false })

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  entries: readonly SlotItem[]
  /** What the button and its menu are called. */
  label: string
  /** The icon of the button; the label is written where there is none. */
  icon?: unknown
  /** The tooltip; the label where none is given. */
  tooltip?: string
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run-command', command: string): void }>()

/* --------------------------------- State ---------------------------------- */
const button = useTemplateRef<HTMLElement>('button')
const isOpen = ref(false)
const trigger = ref<MenuTrigger>('pointer')
const anchor = ref({ x: 0, y: 0 })
const width = ref(0)
const menu = computed(() => createBarMenu(props.entries))
const isMenuOpen = computed(() => isOpen.value && menu.value.items.length > 0)

/* -------------------------------- Handlers -------------------------------- */
function onToggle(event: MouseEvent): void {
  if (isOpen.value) {
    isOpen.value = false
    return
  }
  const box = measureRect(button.value)
  anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0 }
  width.value = box?.width ?? 0
  trigger.value = event.detail === 0 ? 'keyboard' : 'pointer'
  isOpen.value = true
}

function onChoose(id: string): void {
  isOpen.value = false
  const command = menu.value.commands.get(id)
  if (command !== undefined) emit('run-command', command)
}

/* -------------------------------- Helpers --------------------------------- */
function getIcon(id: string): Component | undefined {
  return menu.value.icons.get(id) as Component | undefined
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
    :items="menu.items"
    :at="anchor"
    :asking="width"
    :from="button"
    :trigger="trigger"
    :is-open="isMenuOpen"
    :name="label"
    @choose="onChoose"
    @dismiss="isOpen = false"
  >
    <template #icon="{ id }">
      <component :is="getIcon(id)" v-if="getIcon(id)" :size="16" />
    </template>
  </Menu>
</template>
