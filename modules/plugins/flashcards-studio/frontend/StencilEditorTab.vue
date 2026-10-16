<script setup lang="ts">
/**
 * Standalone stencil editor tab component for Flashcards Studio plugin.
 */
import { computed, ref } from 'vue'
import { StencilEditor } from '@numen/ui'
import type { InsertionPoint, Half } from '@numen/ui'
import type { StencilFace } from './types'

/* ----------------------------- Props & Emits ------------------------------ */
interface StencilTabStateShape {
  stencil?: { value: { fields: readonly string[]; faces: readonly StencilFace[] } }
  errorMessage?: { value: string }
  addField?(name: string): void
  renameField?(field: string, name: string): void
  removeField?(field: string): void
  moveField?(field: string, at: InsertionPoint): void
  addFace?(name: string): void
  renameFace?(id: string, name: string): void
  removeFace?(id: string): void
  moveFace?(face: string, at: InsertionPoint): void
  writeFront?(face: string, prose: string): void
  writeBack?(face: string, prose: string): void
}

const props = defineProps<{
  state?: StencilTabStateShape
  path?: string
  at?: string
}>()

/* --------------------------------- State ---------------------------------- */
const defaultFields = ref<readonly string[]>([])
const defaultFaces = ref<readonly StencilFace[]>([])

const fields = computed(() => props.state?.stencil?.value?.fields ?? defaultFields.value)
const faces = computed(() => props.state?.stencil?.value?.faces ?? defaultFaces.value)

/* -------------------------------- Handlers -------------------------------- */
function onAddField(name: string) {
  props.state?.addField?.(name)
}

function onRenameField(field: string, name: string) {
  props.state?.renameField?.(field, name)
}

function onRemoveField(field: string) {
  props.state?.removeField?.(field)
}

function onMoveField(field: string, at: InsertionPoint) {
  props.state?.moveField?.(field, at)
}

function onAddFace(name: string) {
  props.state?.addFace?.(name)
}

function onRenameFace(id: string, name: string) {
  props.state?.renameFace?.(id, name)
}

function onRemoveFace(id: string) {
  props.state?.removeFace?.(id)
}

function onMoveFace(face: string, at: InsertionPoint) {
  props.state?.moveFace?.(face, at)
}

function onWriteHalf(face: string, half: Half, text: string) {
  if (half === 'front') {
    props.state?.writeFront?.(face, text)
  } else {
    props.state?.writeBack?.(face, text)
  }
}
</script>

<template>
  <div class="stencil-editor-tab">
    <StencilEditor
      :fields="fields"
      :faces="faces"
      @add-field="onAddField"
      @rename-field="onRenameField"
      @remove-field="onRemoveField"
      @move-field="onMoveField"
      @add-face="onAddFace"
      @rename-face="onRenameFace"
      @remove-face="onRemoveFace"
      @move-face="onMoveFace"
      @write-half="onWriteHalf"
    />
  </div>
</template>
