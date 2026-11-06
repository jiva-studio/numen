<script setup lang="ts">
/**
 * The bar at the top of a window: the commands, the title and the window buttons, over an area that
 * moves the window. It draws the commands it is given in the order it is given, and the title opens
 * the list of vaults it is given.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { SearchableListPopup, measureRect, useViewport, type ListItem } from '@numen/ui'
import type { TitleBarViewProps } from '@numen/plugin-sdk'
import { getControls, type ButtonsSetting } from '../lib/frame'
import { formatTitle } from '../lib/title'
import { TITLE_MIN_WIDTH } from '../lib/layout'
import { TITLE_BAR_WORDS } from '../lib/words'
import MacButtonsInset from './frame/MacButtonsInset.vue'
import WindowControls from './frame/WindowControls.vue'
import WindowDragArea from './frame/WindowDragArea.vue'
import TitleBarCommands from './TitleBarCommands.vue'
import './frame/frame.css'
import './titleBar.css'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<
    Partial<TitleBarViewProps> & {
      /** The setting `buttons` of the plugin: where the window buttons stand on Linux. */
      buttons?: ButtonsSetting
    }
  >(),
  {
    pluginId: '',
    appName: '',
    vaultName: '',
    platform: 'windows',
    buttons: 'windows',
    isMaximized: false,
    isFullscreen: false,
    commands: () => [],
    vaults: () => [],
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'minimize-window'): void
  (event: 'toggle-maximize-window'): void
  (event: 'close-window'): void
  (event: 'run-command', id: string): void
  /** A vault that is not the current one was chosen. */
  (event: 'switch-vault', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const controls = computed(() => getControls(props.platform, props.buttons))
const start = useTemplateRef<HTMLElement>('start')
const { viewport } = useViewport(start)
const fullTitle = computed(() => formatTitle(props.appName, props.vaultName))
const room = computed(() =>
  viewport.value.width > 0 ? viewport.value.width - TITLE_MIN_WIDTH : null,
)

const trigger = useTemplateRef<HTMLElement>('trigger')
const isOpen = ref(false)
const anchor = ref({ x: 0, y: 0 })
const triggerWidth = ref(0)
const hasVaults = computed(() => props.vaults.length > 0)
const isListOpen = computed(() => isOpen.value && hasVaults.value)
const items = computed<readonly ListItem[]>(() =>
  props.vaults.map(({ id, name }) => ({ id, text: name })),
)
const current = computed(() => props.vaults.find((one) => one.isCurrent)?.id ?? null)

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
  if (id !== current.value) emit('switch-vault', id)
}

function onPressTitle(event: Event): void {
  if (event.target instanceof Node && trigger.value?.contains(event.target)) {
    event.stopImmediatePropagation()
  }
}

function onDoubleClick(event: MouseEvent): void {
  if (event.target instanceof Element && event.target.closest('button')) return
  emit('toggle-maximize-window')
}
</script>

<template>
  <WindowDragArea
    class="title-bar"
    data-title-bar
    :data-controls="controls"
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

    <div ref="start" class="title-bar__start" data-title-bar-start>
      <TitleBarCommands
        :commands="commands"
        :room="room"
        :words="TITLE_BAR_WORDS"
        @run-command="(id) => emit('run-command', id)"
      />

      <template v-if="hasVaults">
        <button
          ref="trigger"
          type="button"
          class="title-bar__title title-bar__title--trigger"
          data-title
          aria-haspopup="dialog"
          :aria-expanded="isListOpen"
          :title="fullTitle"
          style="--wails-draggable: no-drag"
          @click="onToggleList"
        >
          {{ fullTitle }}
        </button>
        <span class="title-bar__fill" data-title-fill />
      </template>

      <span v-else class="title-bar__title" data-title :title="fullTitle">{{ fullTitle }}</span>
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
