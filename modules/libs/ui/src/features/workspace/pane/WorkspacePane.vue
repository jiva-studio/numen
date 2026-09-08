<script setup lang="ts">
/**
 * A stack of tabs and the one of them that is showing.
 *
 * The strip and the pane carry their identities on the element, so that a
 * drag can find out what the pointer is over by asking the document.
 */
import { inject, onMounted, watch } from 'vue'
import { WorkspaceTab } from '../tab'
import { stepTo } from './keys'
import { WORKSPACE_CONTEXT } from '../context'
import type { Pane, TabId } from '../node'

const props = withDefaults(
  defineProps<{
    pane: Pane
    /** The pane a tab would open into. */
    focused?: boolean
  }>(),
  { focused: false },
)

const workspace = inject(WORKSPACE_CONTEXT)

/** What a tab is called. A tab with no title is shown by its identity. */
const titleOf = (tab: TabId): string => workspace?.value.tabOf(tab)?.title ?? tab

/** What a tab is carrying, and nothing for a tab carrying nothing. */
const markOf = (tab: TabId): string | undefined => workspace?.value.tabOf(tab)?.mark

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
  icon(props: { id: TabId }): unknown
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

/** The tabs as they are drawn, each under the tab it stands for. */
const drawnTabs = new Map<TabId, { focus: () => void }>()

const holdTab = (tab: TabId, drawn: unknown): void => {
  if (drawn) drawnTabs.set(tab, drawn as { focus: () => void })
  else drawnTabs.delete(tab)
}

function reach(tab: TabId): void {
  drawnTabs.get(tab)?.focus()
}

/**
 * The strip is walked with the arrows, and what is reached is shown. The tabs
 * are what it walks; the way to a new tab is a stop of its own.
 */
function along(at: number, event: KeyboardEvent): void {
  const next = stepTo(event.key, at, props.pane.tabs.length)
  const tab = next === null ? undefined : props.pane.tabs[next]
  if (next === null || tab === undefined) return

  event.preventDefault()
  emit('choose', tab)
  reach(tab)
}

/**
 * The way out of what a tab holds, back to the tab itself. A panel that acts
 * on Escape itself keeps it.
 */
function out(event: KeyboardEvent): void {
  if (event.defaultPrevented) return

  const showing = props.pane.active
  if (showing === null) return

  event.preventDefault()
  reach(showing)
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
      class="pane__strip flex min-w-0 shrink-0 items-stretch overflow-hidden"
      role="tablist"
      :data-workspace-strip="pane.id"
    >
      <WorkspaceTab
        v-for="(tab, at) in pane.tabs"
        :id="tabName(at)"
        :ref="(drawn) => holdTab(tab, drawn)"
        :key="tab"
        :aria-controls="panelName(at)"
        :tab="tab"
        :title="titleOf(tab)"
        :mark="markOf(tab)"
        :showing="tab === pane.active"
        :focused="tab === pane.active && focused"
        @lift="emit('lift', tab, $event)"
        @close="emit('close', tab)"
        @click="emit('choose', tab)"
        @keydown="(event: KeyboardEvent) => along(at, event)"
      >
        <template v-if="$slots.icon" #icon>
          <slot name="icon" :id="tab" />
        </template>

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
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.pane__body {
  position: relative;
  overflow: hidden;
}

/* Every tab of the pane is drawn; the ones not shown are held out of sight. What
   a tab holds is alive for as long as the tab is: a caret, a scroll offset, an
   undo history. */
.pane__held {
  display: none;
  block-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
}

.pane__held[data-showing] {
  display: block;
}

/* Drawn inside, because what a pane holds fills it to its edges. */
.pane__held:focus-visible {
  outline: none;
  box-shadow: inset 0 0 0 var(--numen-ring-width) var(--numen-ring);
}

/* The whole of a pane holding no tabs. Its size is all it hands down. */
.pane__silence {
  block-size: 100%;
}

.pane__nothing {
  /* How plainly what is said in place of a tab is drawn. */
  --fade: 0.6;

  display: grid;
  place-items: center;
  block-size: 100%;
  margin: 0;
  opacity: var(--fade);
}
</style>
