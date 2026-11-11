<script setup lang="ts">
/**
 * The single input field of the window: the commands for the current target,
 * and the search.
 *
 * The two share a field, and which one is active determines what is drawn in it
 * and what a keystroke in it means. The keys of the window open and close each.
 */
import { Search } from '@lucide/vue'
import { Palette, type ActionWords } from '@numen/workbench'
import type { CommandDeps } from '@/shared/model/command'
import type { CommandTarget } from '@/shared/model/command'
import type { Commands } from '../model/palette/palette'
import type { ResultChoice, SearchState } from '../model/search/search'
import { findRowIcon, type FileIcons } from '../model/rowIcons'
import { usePaletteInput } from '../model/usePaletteInput'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  commands: Commands
  search: SearchState
  doing: CommandDeps
  getTarget: () => CommandTarget
  places: FileIcons
  runChoice: (choice: ResultChoice) => void
}>()

const words = { ...windowWords, ...PALETTE_WORDS }

/* --------------------------------- State ---------------------------------- */
const actionWords: ActionWords = {
  name: words.actions,
  placeholder: words.findAction,
  silence: words.noAction,
}

const { field, actions } = usePaletteInput({
  commands: props.commands,
  search: props.search,
  doing: props.doing,
  runChoice: props.runChoice,
  findPlaceholder: () => words.find,
})

/* -------------------------------- Handlers -------------------------------- */
function onTyping(text: string) {
  actions.type(text)
}

function onChoose(item: string, action: string) {
  actions.choose(item, action)
}

function onDismiss() {
  actions.dismiss()
}

function onBack() {
  actions.goBack()
}

/* -------------------------------- Helpers --------------------------------- */
function getRowIcon(id: string) {
  return findRowIcon(id, props.commands, props.search, props.places.findFileIcon)
}
</script>

<template>
  <Palette
    :model-value="field.typed"
    :groups="field.groups"
    :is-open="field.open"
    :is-busy="field.isBusy"
    :placeholder="field.placeholder"
    :crumb="field.crumb"
    :step="field.step"
    :opens-on="field.opensOn"
    :name="words.find"
    :action-words="actionWords"
    @update:model-value="onTyping"
    @choose="onChoose"
    @light="commands.palette.previewItem"
    @back="onBack"
    @dismiss="onDismiss"
  >
    <template #icon="{ id }">
      <component :is="getRowIcon(id)" v-if="getRowIcon(id)" class="command-icon" />
    </template>
    <template v-if="!commands.palette.open.value && search.status.isNothingFound.value" #silence>
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
