<script setup lang="ts">
/**
 * The bar every window draws at its top: the actions, the title and the window controls. It names no
 * plugin, no command and no vault; it draws the actions it is given in the order it is given, and
 * the title opens the list of choices it is given.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { SearchableListPopup, measureRect, useViewport, type ListItem } from '@numen/ui'
import { formatTitle } from '../lib/title'
import { NATIVE_MAC_INSET, TITLE_MIN_WIDTH } from '../lib/layout'
import { TITLE_BAR_WORDS } from '../lib/words'
import type { TitleBarProps } from './TitleBar.types'
import TitleBarActions from './TitleBarActions.vue'
import WindowControls from './WindowControls.vue'
import './titleBar.css'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<TitleBarProps>(), {
  vaultName: '',
  controls: 'windows',
  isMaximised: false,
  actions: () => [],
  words: () => ({}),
  choices: () => [],
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'minimise'): void
  (event: 'toggle-maximise'): void
  (event: 'close'): void
  (event: 'run', id: string): void
  /** A choice that is not the current one was chosen. */
  (event: 'choose', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const start = useTemplateRef<HTMLElement>('start')
const { viewport } = useViewport(start)

const fullTitle = computed(() => formatTitle(props.appName, props.vaultName))
const allWords = computed(() => ({ ...TITLE_BAR_WORDS, ...props.words }))
const room = computed(() =>
  viewport.value.width > 0 ? viewport.value.width - TITLE_MIN_WIDTH : null,
)
const insetStyle = { width: `${NATIVE_MAC_INSET}px` }

const trigger = useTemplateRef<HTMLElement>('trigger')
const isOpen = ref(false)
const anchor = ref({ x: 0, y: 0 })
const triggerWidth = ref(0)
const hasChoices = computed(() => props.choices.length > 0)
const isListOpen = computed(() => isOpen.value && hasChoices.value)
const items = computed<readonly ListItem[]>(() =>
  props.choices.map(({ id, label }) => ({ id, text: label })),
)
const current = computed(() => props.choices.find((one) => one.isCurrent)?.id ?? null)

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

function onChoose(id: string): void {
  isOpen.value = false
  if (id !== current.value) emit('choose', id)
}

function onPressTitle(event: Event): void {
  if (event.target instanceof Node && trigger.value?.contains(event.target)) {
    event.stopImmediatePropagation()
  }
}

function onDoubleClick(event: MouseEvent): void {
  if (event.target instanceof Element && event.target.closest('button')) return
  emit('toggle-maximise')
}
</script>

<template>
  <header
    class="title-bar"
    data-title-bar
    :data-controls="controls"
    style="--wails-draggable: drag"
    @dblclick="onDoubleClick"
  >
    <div v-if="controls === 'native-mac'" class="title-bar__inset" data-inset :style="insetStyle" />
    <WindowControls
      v-if="controls === 'mac'"
      look="mac"
      :is-maximised="isMaximised"
      :words="allWords"
      @minimise="emit('minimise')"
      @toggle-maximise="emit('toggle-maximise')"
      @close="emit('close')"
    />
    <div ref="start" class="title-bar__start" data-title-bar-start>
      <TitleBarActions
        :actions="actions"
        :room="room"
        :words="allWords"
        @run="(id) => emit('run', id)"
      />
      <template v-if="hasChoices">
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
      :label="allWords.switcher"
      :placeholder="allWords.search"
      :empty-text="allWords.nothing"
      :at="anchor"
      :asking="triggerWidth"
      :from="trigger"
      @select="onChoose"
      @close="isOpen = false"
    />
    <WindowControls
      v-if="controls === 'windows'"
      look="windows"
      :is-maximised="isMaximised"
      :words="allWords"
      @minimise="emit('minimise')"
      @toggle-maximise="emit('toggle-maximise')"
      @close="emit('close')"
    />
  </header>
</template>
