<script setup lang="ts">
/**
 * A stack of tabs and the one of them that is showing.
 *
 * The strip and the pane carry their identities on the element, so that a
 * drag can find out what the pointer is over by asking the document.
 */
import { onMounted, useTemplateRef, watch } from 'vue'
import WorkspaceTab from './WorkspaceTab.vue'
import { stepTo } from './keys'
import type { Pane, TabId } from '../model'

const props = withDefaults(
  defineProps<{
    pane: Pane
    /** What each tab is called. A tab with no title is shown by its identity. */
    titles: Readonly<Record<TabId, string>>
    /** What a tab is carrying, for the tabs that are carrying anything. */
    marks?: Readonly<Record<TabId, string>>
    /** The pane a tab would open into. */
    focused?: boolean
  }>(),
  { marks: () => ({}), focused: false },
)

const emit = defineEmits<{
  (event: 'choose', tab: TabId): void
  (event: 'close', tab: TabId): void
  (event: 'lift', tab: TabId, at: PointerEvent): void
  /** This pane asks to be the one a tab opens into. */
  (event: 'claim'): void
  /** The tab that is now the one showing, once it is on screen. */
  (event: 'show', tab: TabId): void
}>()

defineSlots<{
  tab(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/** A pane is claimed under the primary button and under no other. */
function claim(event: PointerEvent): void {
  if (event.button === 0) emit('claim')
}

/**
 * A tab held out of sight is drawn with no size, so what is showing is said
 * once it is on screen and can be measured. It is said for the tab this pane
 * opens with as well.
 */
onMounted(() => {
  if (props.pane.active !== null) emit('show', props.pane.active)
})

watch(
  () => props.pane.active,
  (tab) => {
    if (tab !== null) emit('show', tab)
  },
  { flush: 'post' },
)

/**
 * A tab and the panel it stands over name each other, so the names are unique
 * to the document and not only to this pane.
 */
const tabName = (at: number): string => `${props.pane.id}-tab-${at}`
const panelName = (at: number): string => `${props.pane.id}-panel-${at}`

const strip = useTemplateRef<HTMLElement>('strip')

function reach(at: number): void {
  strip.value?.querySelectorAll<HTMLElement>('[data-workspace-tab]')[at]?.focus()
}

/**
 * The strip is walked with the arrows, and what is reached is shown. The tabs
 * are what it walks; the way to a new tab is a stop of its own.
 */
function along(event: KeyboardEvent): void {
  const held = (event.target as Element | null)
    ?.closest('[data-workspace-tab]')
    ?.getAttribute('data-workspace-tab')
  if (!held) return

  const next = stepTo(event.key, props.pane.tabs.indexOf(held), props.pane.tabs.length)
  const tab = next === null ? undefined : props.pane.tabs[next]
  if (next === null || tab === undefined) return

  event.preventDefault()
  emit('choose', tab)
  reach(next)
}

/**
 * The way out of what a tab holds, back to the tab itself. A panel that acts
 * on Escape itself keeps it.
 */
function out(event: KeyboardEvent): void {
  if (event.defaultPrevented) return

  const at = props.pane.active === null ? -1 : props.pane.tabs.indexOf(props.pane.active)
  if (at < 0) return

  event.preventDefault()
  reach(at)
}
</script>

<template>
  <section
    class="pane numen flex min-h-0 min-w-0 flex-col bg-surface text-ink"
    :data-workspace-pane="pane.id"
    :data-focused="focused || undefined"
    @pointerdown="claim"
  >
    <!-- The strip is the list of tabs. A pane holding no tabs has none. -->
    <div
      v-if="pane.tabs.length > 0"
      ref="strip"
      class="pane__strip flex min-w-0 shrink-0 items-stretch overflow-hidden"
      role="tablist"
      :data-workspace-strip="pane.id"
      @keydown="along"
    >
      <WorkspaceTab
        v-for="(tab, at) in pane.tabs"
        :id="tabName(at)"
        :key="tab"
        :aria-controls="panelName(at)"
        :tab="tab"
        :title="titles[tab] ?? tab"
        :mark="marks[tab]"
        :showing="tab === pane.active"
        :focused="tab === pane.active && focused"
        @lift="emit('lift', tab, $event)"
        @close="emit('close', tab)"
        @click="emit('choose', tab)"
      >
        <template v-if="$slots.mark" #mark="bound">
          <slot name="mark" :id="tab" v-bind="bound" />
        </template>
      </WorkspaceTab>
    </div>

    <div class="pane__body min-h-0 min-w-0 flex-1">
      <div
        v-for="(tab, at) in pane.tabs"
        :id="panelName(at)"
        :key="tab"
        class="pane__held"
        role="tabpanel"
        tabindex="0"
        :aria-labelledby="tabName(at)"
        :data-showing="tab === pane.active || undefined"
        @keydown.escape="out"
      >
        <slot name="tab" :id="tab" />
      </div>
      <div v-if="pane.tabs.length === 0" class="pane__silence">
        <slot name="silence">
          <p class="pane__nothing font-sans text-small text-hushed">Nothing open</p>
        </slot>
      </div>
    </div>
  </section>
</template>

<style scoped>
.pane {
  block-size: 100%;
}

/* Sits on the same surface as what it stands over, told apart by one line. */
.pane__strip {
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
}

.pane__body {
  position: relative;
  overflow: hidden;
}

/* Every tab of the pane is drawn; the ones not shown are held out of sight.
 *
 * What a tab holds is alive for as long as the tab is: a caret, a scroll offset,
 * an undo history. Drawing only the active one hands those back to the person
 * emptied every time they look at something else. */
.pane__held {
  display: none;
  block-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
}

.pane__held[data-showing] {
  display: block;
}

/* The whole of a pane holding no tabs. Its size is all it hands down. */
.pane__silence {
  block-size: 100%;
}

.pane__nothing {
  display: grid;
  place-items: center;
  block-size: 100%;
  margin: 0;
  opacity: 0.6;
}
</style>
