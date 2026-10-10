<script setup lang="ts">
/**
 * Displays the graph neighborhood of notes and handles graph interaction gestures.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { Menu, optionsForType, Plex, threadOf, useTypeSize, Waiting } from '@numen/ui'
import type { MenuOpening, PlexRelatedSeat, PlexDestination } from '@numen/ui'
import type { LucideIcon } from '@lucide/vue'
import PlexQuickLinkPopover from './PlexQuickLinkPopover.vue'
import { ITEMS, NONE } from '../lib/menu'
import { iconFor } from '@/shared/icons'
import { iconOfNote } from '@/entities/note'
import type { PlexTabState } from '../types'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: PlexTabState }>()

/* --------------------------------- State ---------------------------------- */
// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const { dragged, empty, menu, quickLink, mostParts, picture: neighbourhood } = props.state

/**
 * How large the picture is drawn, and how many parts a node hangs at once. A
 * node's label is set in the window's type, so the boxes and the clearances
 * between them are handed in to hold it.
 */
const type = useTypeSize()
const options = computed(() => ({
  ...optionsForType(type.value),
  maxParts: mostParts.value,
}))

/** What the menu offers: on a node, or off every node. */
const items = computed(() => (menu.value?.node === null ? NONE : ITEMS))

const lastPointer = ref<{ x: number; y: number }>({ x: 300, y: 300 })

const rootElement = useTemplateRef<HTMLElement>('rootElement')
const picture = useTemplateRef<{ focusNode: (id: string) => void }>('picture')

const sourceCenter = computed(() => {
  if (!quickLink.value || !rootElement.value) return null
  return getSourceHandlePosition(
    rootElement.value,
    quickLink.value.from,
    props.state.picture.value?.nodes,
  )
})

const threadPath = computed(() => {
  if (!sourceCenter.value || !quickLink.value) return ''
  return threadOf(sourceCenter.value, quickLink.value.at)
})

/* -------------------------------- Handlers -------------------------------- */
/**
 * A menu asked for over a tab drawing no picture, which is a vault holding no
 * note to draw one around. A tab drawing one leaves the picture to answer, and
 * so does a vault that has notes and has not been read yet.
 */
function onContextMenu(event: MouseEvent) {
  if (!empty.value) return
  event.preventDefault()
  props.state.openMenu({
    node: null,
    at: { x: event.clientX, y: event.clientY },
    opening: 'pointer',
  })
}

function onActivateNode(node: string) {
  props.state.activate(node)
}

function onPointerUp(event: PointerEvent) {
  const rect = rootElement.value?.getBoundingClientRect()
  if (rect) {
    lastPointer.value = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    }
  } else {
    lastPointer.value = { x: event.clientX, y: event.clientY }
  }
}

function onCreateNode(from: string, seat: PlexRelatedSeat, at?: { x: number; y: number }) {
  void props.state.createNode(from, seat, at ?? lastPointer.value)
}

function onLinkNodes(from: string, to: string, seat: PlexRelatedSeat) {
  void props.state.joinNodes(from, to, seat)
}

function onBringNodes(nodes: readonly string[], seat: PlexRelatedSeat) {
  void props.state.dropNodes(nodes, seat)
}

function onOpenMenu(node: string, at: { x: number; y: number }, opening: MenuOpening) {
  props.state.openMenu({ node, at, opening })
}

function onShowNode(node: string, how: PlexDestination) {
  props.state.openNode(node, how)
}

function onEnterPart(node: string, part: string) {
  props.state.openPart(node, part)
}

function onDismissPicture() {
  props.state.dismiss()
}

function onChooseMenuItem(id: string) {
  closeMenu(id)
}

function onDismissMenu() {
  closeMenu()
}

function onSelectQuickLinkNote(path: string) {
  void props.state.confirmQuickLink(path, true)
}

function onCreateQuickLinkNote(title: string) {
  void props.state.confirmQuickLink(title, false)
}

function onDismissQuickLink() {
  props.state.dismissQuickLink()
}

/* -------------------------------- Helpers --------------------------------- */
/**
 * What a node is drawn before its title, and nothing for an ordinary note. A
 * deck, a stencil and a preset carry the icon the tree draws them under.
 */
function getNodeIcon(node: string): LucideIcon | null {
  const type = props.state.typeOf(node)
  return type === 'note' ? null : iconOfNote(type)
}

/** A menu put away, and the keyboard back on the node it was asked from. */
function closeMenu(id?: string) {
  const node = menu.value?.node ?? null
  if (id === undefined) props.state.dismiss()
  else props.state.chooseMenuItem(id)
  if (node !== null) picture.value?.focusNode(node)
}

function findSourceNodeElement(svg: SVGSVGElement, title: string): Element | null {
  for (const el of svg.querySelectorAll('g.plex__node')) {
    const label = el.getAttribute('aria-label')
    if (label && title && label.startsWith(title)) return el
  }
  return svg.querySelector('g.plex__node--focus')
}

function getHandleFromCTM(
  boxEl: Element | null,
  transform: string,
  svgEl: SVGSVGElement,
  rootRect: DOMRect,
): { x: number; y: number } | null {
  const width = Number(boxEl?.getAttribute('width') || 0)
  const match = /translate\(\s*([-\d.]+)\s+([-\d.]+)\s*\)/.exec(transform)
  if (!match?.[1] || !match[2] || width <= 0) return null
  const ctm = svgEl.getScreenCTM?.()
  if (!ctm) return null
  const nodeX = parseFloat(match[1]) + width / 2
  const nodeY = parseFloat(match[2])
  return {
    x: ctm.a * nodeX + ctm.c * nodeY + ctm.e - rootRect.left,
    y: ctm.b * nodeX + ctm.d * nodeY + ctm.f - rootRect.top,
  }
}

function getSourceHandlePosition(
  root: HTMLElement,
  fromId: string,
  nodes?: readonly { readonly id: string; readonly title: string }[],
): { x: number; y: number } | null {
  const sourceNode = nodes?.find((n) => n.id === fromId)
  const svgEl = root.querySelector('svg.plex') as SVGSVGElement | null
  if (!sourceNode || !svgEl) return null

  const nodeEl = findSourceNodeElement(svgEl, sourceNode.title)
  if (!nodeEl) return null

  const boxEl = nodeEl.querySelector('.plex__box')
  const rootRect = root.getBoundingClientRect()
  const ctmPos = getHandleFromCTM(boxEl, nodeEl.getAttribute('transform') || '', svgEl, rootRect)
  if (ctmPos) return ctmPos

  const rect = (boxEl ?? nodeEl).getBoundingClientRect()
  return {
    x: rect.right - rootRect.left,
    y: rect.top + rect.height / 2 - rootRect.top,
  }
}
</script>

<template>
  <div ref="rootElement" class="plex" @contextmenu="onContextMenu" @pointerup.capture="onPointerUp">
    <p v-if="props.state.view.error.value" class="caution">
      {{ props.state.view.error.value }}
    </p>

    <!-- Until the first neighbourhood lands, an empty plex and one nobody has
         answered for yet are drawn the same way. -->
    <Waiting v-if="props.state.view.isLoading.value" :label="words.loading" />

    <Plex
      v-else-if="neighbourhood"
      ref="picture"
      class="plex__picture"
      :neighbourhood="neighbourhood!"
      :options="options"
      :creatable="props.state.creatable"
      :dragged="dragged"
      :drop-name="words.dropName"
      :parts="props.state.partsOf"
      @activate="onActivateNode"
      @create="onCreateNode"
      @link="onLinkNodes"
      @bring="onBringNodes"
      @menu="onOpenMenu"
      @show="onShowNode"
      @enter="onEnterPart"
      @dismiss="onDismissPicture"
    >
      <!-- A deck, a stencil and a preset are drawn as the tree draws them. An
           ordinary note is drawn its title and nothing before it. -->
      <template #icon="{ node }: { node: { id: string } }">
        <component
          :is="getNodeIcon(node.id)"
          v-if="getNodeIcon(node.id)"
          class="plex__icon"
          aria-hidden="true"
        />
      </template>
    </Plex>

    <Menu
      v-if="menu"
      :items="items"
      :at="menu!.at"
      :opening="menu!.opening"
      open
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
    >
      <template #icon="{ id }">
        <component :is="iconFor(id)" v-if="iconFor(id)" class="plex__icon" aria-hidden="true" />
      </template>
    </Menu>

    <svg v-if="quickLink && threadPath" class="plex__quick-link-thread" aria-hidden="true">
      <path :d="threadPath" class="plex__thread" />
    </svg>

    <PlexQuickLinkPopover
      v-if="quickLink"
      :at="quickLink.at"
      :seat="quickLink.seat"
      :search="props.state.searchNotes"
      @select-note="onSelectQuickLinkNote"
      @create-note="onCreateQuickLinkNote"
      @dismiss="onDismissQuickLink"
    />
  </div>
</template>

<style scoped>
/* The picture takes what the band above it leaves. */
.plex {
  position: relative;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  overflow: hidden;
}

.plex__picture {
  flex: 1;
  min-block-size: 0;
}

.plex__quick-link-thread {
  position: absolute;
  inset: 0;
  inline-size: 100%;
  block-size: 100%;
  pointer-events: none;
  z-index: 10;
}

.plex__quick-link-thread .plex__thread {
  fill: none;
  stroke: var(--numen-edge);
  stroke-width: var(--numen-edge-width);
  stroke-dasharray: var(--numen-thread-dash);
}

/* Lucide draws on a 24 grid, and the stroke is given in those units.
   A node's title stands in a `foreignObject`, and nothing drawn in one is given
   an opacity of its own: WebKit paints what it makes a layer of at the corner
   of the picture. Anything quieter is asked for in the colour. */
.plex__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
}
</style>
