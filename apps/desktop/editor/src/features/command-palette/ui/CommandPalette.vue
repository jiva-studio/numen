<script setup lang="ts">
/**
 * The single input field of the window: the commands for the current target,
 * and the search.
 *
 * The two share a field, and which one is active determines what is drawn in it
 * and what a keystroke in it means. The keys of the window open and close each.
 */
import { Search } from '@lucide/vue'
import { reportFailure, runInBackground } from '@numen/ui'
import { Palette, type ActionWords } from '@numen/workbench'
import { findRowIcon } from '../model/rowIcons'
import { usePaletteInput } from '../model/usePaletteInput'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../words'
import type { CommandPaletteProps } from './CommandPalette.types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<CommandPaletteProps>()

const words = { ...windowWords, ...PALETTE_WORDS }

/* --------------------------------- State ---------------------------------- */
const actionWords: ActionWords = {
  name: words.actions,
  placeholder: words.findAction,
  silence: words.noAction,
}

const { field, type, choose, dismiss, goBack } = usePaletteInput({
  commands: props.commands,
  search: props.search,
  doing: props.doing,
  runChoice: props.runChoice,
  findPlaceholder: () => words.find,
})

/* -------------------------------- Handlers -------------------------------- */
function onType(text: string): void {
  if (props.picker.question.value) props.picker.question.value.enterText(text)
  else type(text)
}

function onChoose(item: string, action: string): void {
  if (props.picker.question.value) props.picker.question.value.answer(item)
  else choose(item, action)
}

function onDismiss(): void {
  if (props.picker.multi.value || props.picker.question.value) props.picker.dismiss()
  else dismiss()
}

/* -------------------------------- Helpers --------------------------------- */
function getRowIcon(id: string) {
  return findRowIcon(id, props.commands, props.search, props.places.findFileIcon)
}
</script>

<template>
  <Palette
    :model-value="picker.question.value?.typed ?? field.typed"
    :groups="picker.question.value?.groups ?? field.groups"
    :is-open="field.open || picker.multi.value !== undefined || picker.question.value !== undefined"
    :multi="picker.multi.value"
    :is-busy="field.isBusy"
    :placeholder="picker.question.value?.placeholder ?? field.placeholder"
    :heading="picker.question.value?.heading"
    :crumb="field.crumb"
    :step="picker.question.value?.step ?? field.step"
    :opens-on="field.opensOn"
    :name="words.find"
    :action-words="actionWords"
    @update:model-value="onType"
    @choose="onChoose"
    @submit="picker.question.value?.submit()"
    @light="commands.palette.previewItem"
    @back="goBack"
    @toggle="runInBackground(picker.toggle($event), reportFailure)"
    @create="runInBackground(picker.create($event), reportFailure)"
    @dismiss="onDismiss"
  >
    <template #icon="{ id }">
      <component :is="getRowIcon(id)" v-if="getRowIcon(id)" class="command-icon" />
    </template>
    <template
      v-if="
        !picker.question.value && !commands.palette.open.value && search.status.isNothingFound.value
      "
      #silence
    >
      <span class="nothing-found">
        <Search class="nothing-found__icon" data-palette="nothing-icon" aria-hidden="true" />
        <span class="nothing-found__text" data-palette="nothing-text">
          {{ words.nothingFound(search.typed.value.trim()) }}
        </span>
      </span>
    </template>
  </Palette>
</template>

<style scoped>
.nothing-found {
  --gap: 0.5rem;
  --pad-block: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: var(--gap);
  align-items: center;
  padding-block: var(--pad-block);
  text-align: center;
}

.nothing-found__icon {
  --size: 1.5rem;
  flex: none;
  inline-size: var(--size);
  block-size: var(--size);
  stroke-width: 1.75;
}

/* A long query is cut by the edge of the panel. */
.nothing-found__text {
  max-inline-size: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.command-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}
</style>
