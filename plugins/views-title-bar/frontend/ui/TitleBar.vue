<script setup lang="ts">
/**
 * The bar at the top of a window: the entries of `titleBar.commands`, the title, the entries of
 * `titleBar.status` and the window buttons, over an area that moves the window. The title opens the
 * list of vaults. When the room is short, the commands go into the menu button and the views are
 * hidden from the end of the bar.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { SearchableListPopup, measureRect, type ListItem } from '@numen/ui'
import {
  SLOT,
  type AppInfo,
  type CommandItem,
  type SlotItem,
  type VaultsState,
} from '@numen/plugin-sdk'
import { getControls, getControlsRoom, type ButtonsSetting } from '../lib/frame'
import { formatTitle } from '../lib/title'
import { TITLE_BAR_WORDS } from '../lib/words'
import MacButtonsInset from './frame/MacButtonsInset.vue'
import WindowControls from './frame/WindowControls.vue'
import WindowDragArea from './frame/WindowDragArea.vue'
import TitleBarMenuButton from './TitleBarMenuButton.vue'
import TitleBarSlot from './TitleBarSlot.vue'
import { getFitKey, useBarFit, type BarRow } from './useBarFit'
import { useToolbarFocus } from './useToolbarFocus'
import './frame/frame.css'
import './titleBar.css'

/** The id of the last row of the list; its NUL character keeps it apart from every vault id. */
const CREATE_VAULT_ROW = '\u0000create-vault'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    appName: string
    platform?: AppInfo['platform']
    buttons?: ButtonsSetting
    isMaximized?: boolean
    isFullscreen?: boolean
    vaults?: VaultsState
    commands?: readonly SlotItem[]
    status?: readonly SlotItem[]
    canCreateVault?: boolean
    listItems?: (command: string) => Promise<readonly CommandItem[]>
    reportError?: ((error: unknown) => void) | undefined
  }>(),
  {
    platform: 'windows',
    buttons: 'windows',
    isMaximized: false,
    isFullscreen: false,
    vaults: () => ({ vaults: [], current: null }),
    commands: () => [],
    status: () => [],
    canCreateVault: false,
    listItems: async () => [],
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'minimize-window'): void
  (event: 'toggle-maximize-window'): void
  (event: 'close-window'): void
  /** A command was picked; with `itemId`, the id of the item picked. */
  (event: 'run-command', command: string, itemId?: string): void
  /** A vault that is not the current one was chosen. */
  (event: 'switch-vault', id: string): void
  /** The last row of the list, which creates a vault, was chosen. */
  (event: 'create-vault'): void
}>()

/* --------------------------------- State ---------------------------------- */
const controls = computed(() => getControls(props.platform, props.buttons))
const start = useTemplateRef<HTMLElement>('start')
const fullTitle = computed(() => formatTitle(props.appName, props.vaults.current?.name ?? ''))

const rows = computed<readonly BarRow[]>(() => [
  { prefix: SLOT.titleBarCommands, entries: props.commands },
  { prefix: SLOT.titleBarStatus, entries: props.status },
])
const room = computed(() => getControlsRoom(controls.value, props.isFullscreen))
const { fit } = useBarFit(start, () => rows.value, () => room.value.lead + room.value.trail)
const { onKeydown, onFocusin } = useToolbarFocus(start)
const overflow = computed<readonly SlotItem[]>(() => {
  const overflowKeys = new Set(fit.value.overflow)
  return rows.value.flatMap(({ prefix, entries }, index) => [
    ...(index > 0 ? [{ kind: 'separator', key: `${prefix}#start` } as const] : []),
    ...entries.filter(
      (entry) => entry.kind === 'separator' || overflowKeys.has(getFitKey(prefix, entry)),
    ),
  ])
})

const trigger = useTemplateRef<HTMLElement>('trigger')
const isOpen = ref(false)
const anchor = ref({ x: 0, y: 0 })
const triggerWidth = ref(0)
const hasVaults = computed(() => props.vaults.current !== null && props.vaults.vaults.length > 0)
const isListOpen = computed(() => isOpen.value && hasVaults.value)
const items = computed<readonly ListItem[]>(() => [
  ...props.vaults.vaults.map(({ id, name }) => ({ id, text: name })),
  ...(props.canCreateVault
    ? [{ id: CREATE_VAULT_ROW, text: TITLE_BAR_WORDS.createVault, hasSeparator: true }]
    : []),
])
const current = computed(() => props.vaults.current?.id ?? null)

/* --------------------------------- Hooks ---------------------------------- */
// Registered before the list is drawn, so it runs before the list's own press outside: a press of
// the title is the title's to answer, which closes the list through the click that follows.
watch(
  isListOpen,
  (open) => {
    if (open) globalThis.addEventListener('pointerdown', onPressTitle, true)
    else globalThis.removeEventListener('pointerdown', onPressTitle, true)
  },
  { flush: 'sync' },
)

onBeforeUnmount(() => globalThis.removeEventListener('pointerdown', onPressTitle, true))

/* -------------------------------- Handlers -------------------------------- */
function onToggleList(): void {
  if (isOpen.value) {
    isOpen.value = false
    return
  }
  const box = measureRect(trigger.value)
  anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0 }
  triggerWidth.value = box?.width ?? 0
  isOpen.value = true
}

function onSwitchVault(id: string): void {
  isOpen.value = false
  if (id === CREATE_VAULT_ROW) emit('create-vault')
  else if (id !== current.value) emit('switch-vault', id)
}

function onPressTitle(event: Event): void {
  if (event.target instanceof Node && trigger.value?.contains(event.target)) {
    event.stopImmediatePropagation()
  }
}

function onDoubleClick(event: MouseEvent): void {
  if (event.target instanceof Element && event.target.closest('button, [data-title-bar-entry]')) {
    return
  }
  emit('toggle-maximize-window')
}

function onRunCommand(command: string, itemId?: string): void {
  if (itemId === undefined) emit('run-command', command)
  else emit('run-command', command, itemId)
}
</script>

<template>
  <WindowDragArea
    class="title-bar"
    data-title-bar
    :data-controls="controls"
    :style="{ '--title-bar-lead': `${room.lead}px`, '--title-bar-trail': `${room.trail}px` }"
    @dblclick="onDoubleClick"
  >
    <MacButtonsInset
      v-if="controls === 'native-mac'"
      class="title-bar__inset"
      :is-fullscreen="isFullscreen"
    />

    <WindowControls
      v-if="controls === 'mac'"
      look="mac"
      :is-maximized="isMaximized"
      :words="TITLE_BAR_WORDS"
      @minimize-window="emit('minimize-window')"
      @toggle-maximize-window="emit('toggle-maximize-window')"
      @close-window="emit('close-window')"
    />

    <div
      ref="start"
      class="title-bar__start"
      data-title-bar-start
      role="toolbar"
      :aria-label="TITLE_BAR_WORDS.toolbar"
      @keydown="onKeydown"
      @focusin="onFocusin"
    >
      <div class="title-bar__lead" data-title-bar-lead>
        <TitleBarSlot
          :name="SLOT.titleBarCommands"
          :entries="commands"
          :get-key="(entry) => getFitKey(SLOT.titleBarCommands, entry)"
          :hidden="fit.hidden"
          :platform="platform"
          :list-items="listItems"
          :report-error="reportError"
          @run-command="onRunCommand"
        />

        <TitleBarMenuButton
          v-if="fit.overflow.length > 0"
          :entries="overflow"
          :label="TITLE_BAR_WORDS.more"
          :icon="Ellipsis"
          data-title-bar-more
          :list-items="listItems"
          :report-error="reportError"
          @run-command="onRunCommand"
        />
      </div>

      <div class="title-bar__middle" data-title-bar-middle>
        <button
          v-if="hasVaults"
          ref="trigger"
          type="button"
          class="title-bar__title title-bar__title--trigger"
          data-title
          data-title-bar-focusable
          aria-haspopup="dialog"
          :aria-expanded="isListOpen"
          :title="fullTitle"
          style="--wails-draggable: no-drag"
          @click="onToggleList"
        >
          {{ fullTitle }}
        </button>
        <span v-else class="title-bar__title" data-title :title="fullTitle">{{ fullTitle }}</span>
      </div>

      <div class="title-bar__trail" data-title-bar-trail>
        <TitleBarSlot
          :name="SLOT.titleBarStatus"
          :entries="status"
          :get-key="(entry) => getFitKey(SLOT.titleBarStatus, entry)"
          :hidden="fit.hidden"
          :platform="platform"
          :list-items="listItems"
          :report-error="reportError"
          @run-command="onRunCommand"
        />
      </div>
    </div>

    <SearchableListPopup
      v-if="isListOpen"
      :items="items"
      :model-value="current"
      :label="TITLE_BAR_WORDS.switcher"
      :placeholder="TITLE_BAR_WORDS.search"
      :empty-text="TITLE_BAR_WORDS.nothing"
      :at="anchor"
      :asking="triggerWidth"
      :from="trigger"
      @select="onSwitchVault"
      @close="isOpen = false"
    />

    <WindowControls
      v-if="controls === 'windows'"
      look="windows"
      :is-maximized="isMaximized"
      :words="TITLE_BAR_WORDS"
      @minimize-window="emit('minimize-window')"
      @toggle-maximize-window="emit('toggle-maximize-window')"
      @close-window="emit('close-window')"
    />
  </WindowDragArea>
</template>
