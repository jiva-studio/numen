<script setup lang="ts">
/**
 * Standalone deck editor tab component for Flashcards Studio plugin.
 */
import { computed, ref } from 'vue'
import { DeckEditor } from '@numen/ui'
import type { DeckCard, DeckSection, Stencil, InsertionPoint } from '@numen/ui'

/* ----------------------------- Props & Emits ------------------------------ */
interface DeckTabStateShape {
  drawn?: { value: readonly DeckCard[] }
  sections?: { value: readonly DeckSection[] }
  stencils?: { value: readonly Stencil[] }
  errorMessage?: { value: string }
  addCard?(stencil: string, values: unknown, section: string | null): void
  removeCard?(id: string): void
  moveCard?(id: string, at: InsertionPoint): void
  writeCardField?(id: string, field: string, nth: number, text: string): void
  addSection?(name: string): void
  renameSection?(id: string, name: string): void
  removeSection?(id: string): void
}

const props = defineProps<{
  state?: DeckTabStateShape
  path?: string
  at?: string
}>()

/* --------------------------------- State ---------------------------------- */
const internalCards = ref<readonly DeckCard[]>([])
const internalSections = ref<readonly DeckSection[]>([])
const internalStencils = ref<readonly Stencil[]>([])

const cards = computed(() => props.state?.drawn?.value ?? internalCards.value)
const sections = computed(() => props.state?.sections?.value ?? internalSections.value)
const stencils = computed(() => props.state?.stencils?.value ?? internalStencils.value)

/* -------------------------------- Handlers -------------------------------- */
function onAddCard(stencil: string, section: string | null) {
  props.state?.addCard?.(stencil, [], section)
}

function onRemoveCard(id: string) {
  props.state?.removeCard?.(id)
}

function onMoveCard(id: string, at: InsertionPoint) {
  props.state?.moveCard?.(id, at)
}

function onWriteCard(id: string, field: string, nth: number, text: string) {
  props.state?.writeCardField?.(id, field, nth, text)
}

function onAddSection(name: string) {
  props.state?.addSection?.(name)
}

function onRenameSection(id: string, name: string) {
  props.state?.renameSection?.(id, name)
}

function onRemoveSection(id: string) {
  props.state?.removeSection?.(id)
}
</script>

<template>
  <div class="deck-editor-tab">
    <DeckEditor
      :cards="cards"
      :sections="sections"
      :stencils="stencils"
      @add="onAddCard"
      @remove="onRemoveCard"
      @move="onMoveCard"
      @write="onWriteCard"
      @add-section="onAddSection"
      @rename-section="onRenameSection"
      @remove-section="onRemoveSection"
    />
  </div>
</template>
