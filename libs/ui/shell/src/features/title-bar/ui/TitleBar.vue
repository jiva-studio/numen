<script setup lang="ts">
/**
 * The bar every window draws at its top: the actions, the title and the window controls. It names no
 * plugin and no command; it draws the actions it is given in the order it is given.
 */
import { computed, useTemplateRef } from 'vue'
import { useViewport } from '@numen/ui'
import { formatTitle } from '../lib/title'
import { NATIVE_MAC_INSET, TITLE_MIN_WIDTH } from '../model/layout'
import type { ControlsStyle, TitleBarAction } from '../model/types'
import { TITLE_BAR_WORDS, type TitleBarWords } from '../model/words'
import TitleBarActions from './TitleBarActions.vue'
import WindowControls from './WindowControls.vue'
import './titleBar.css'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    appName: string
    /** The vault in the window; empty where none is open. */
    vaultName?: string
    controls?: ControlsStyle
    maximised?: boolean
    /** Sorted as they are to be drawn. */
    actions?: readonly TitleBarAction[]
    words?: Partial<TitleBarWords>
  }>(),
  {
    vaultName: '',
    controls: 'windows',
    maximised: false,
    actions: () => [],
    words: () => ({}),
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'minimise'): void
  (event: 'toggle-maximise'): void
  (event: 'close'): void
  (event: 'run', id: string): void
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

/* -------------------------------- Handlers -------------------------------- */
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
      :maximised="maximised"
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
      <span class="title-bar__title" data-title :title="fullTitle">{{ fullTitle }}</span>
    </div>
    <WindowControls
      v-if="controls === 'windows'"
      look="windows"
      :maximised="maximised"
      :words="allWords"
      @minimise="emit('minimise')"
      @toggle-maximise="emit('toggle-maximise')"
      @close="emit('close')"
    />
  </header>
</template>
