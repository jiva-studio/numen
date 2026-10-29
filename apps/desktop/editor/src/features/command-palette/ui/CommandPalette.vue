<script setup lang="ts">
/**
 * The single input field of the window: the commands for the current target,
 * and the search.
 *
 * The two share a field, and which one is active determines what is drawn in it
 * and what a keystroke in it means. Two chords open and close each.
 */
import { computed } from 'vue'
import { Palette, type ActionWords } from '@numen/shell'
import { isCommandPrefixTyped } from '../lib/commands/commands'
import type { CommandDeps } from '../model/handlers/deps'
import type { CommandTarget } from '../types'
import { appendCreateNoteItem, createNoteInvocation, CREATE_NOTE_ID } from '../lib/offers/offers'
import type { Commands } from '../model/palette/palette'
import { runInvocation } from '../model/handlers/handlers'
import type { SearchState } from '../model/search/search'
import { usePaletteShortcuts } from '../model/palette/shortcuts'
import { findRowIcon } from '../model/rowIcons'
import { openDestination, type DestinationDeps } from '../model/search/destination'
import { ANSWER_WORDS } from '../words'
import { WORDS as words } from '@/shared/words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  commands: Commands
  search: SearchState
  doing: CommandDeps
  /** The target the commands apply to, as the window is now. */
  getTarget: () => CommandTarget
  /** Where the window is taken by what the search turns up. */
  places: DestinationDeps
}>()

/* --------------------------------- State ---------------------------------- */
const actionWords: ActionWords = {
  name: words.actions,
  placeholder: words.findAction,
  silence: words.noAction,
}

usePaletteShortcuts(props.search, props.commands)

/** What the palette draws: the commands while they are open, the search under. */
const field = computed(() =>
  props.commands.open.value
    ? {
        open: true,
        typed: props.commands.typed.value,
        groups: props.commands.groups.value,
        crumb: props.commands.crumb.value,
        step: props.commands.step.value,
        opensOn: props.commands.opensOn.value,
        placeholder: props.commands.placeholder.value,
      }
    : {
        open: props.search.open.value,
        typed: props.search.typed.value,
        groups: appendCreateNoteItem(props.search.groups.value, {
          text: props.search.typed.value,
          words,
          target: props.getTarget(),
        }),
        crumb: '',
        step: '',
        opensOn: '',
        placeholder: words.find,
      },
)

/* -------------------------------- Handlers -------------------------------- */
function onTyping(text: string) {
  if (props.commands.open.value) return void props.commands.setTyped(text)
  if (!isCommandPrefixTyped(props.search.typed.value, text)) return void props.search.setTyped(text)
  const setSearchOpen = props.search.setOpen
  const setCommandsOpen = props.commands.setOpen
  setSearchOpen(false)
  setCommandsOpen(true)
}

async function onChoose(item: string, action: string) {
  const setSearchOpen = props.search.setOpen
  const setCommandsOpen = props.commands.setOpen
  if (props.commands.open.value) {
    const invocation = props.commands.chooseItem(item, action)
    if (!invocation) return
    setCommandsOpen(false)
    await runInvocation(invocation, props.doing, ANSWER_WORDS)
    return
  }
  if (item === CREATE_NOTE_ID) {
    const name = props.search.typed.value.trim()
    setSearchOpen(false)
    await runInvocation(
      createNoteInvocation(action, name, props.getTarget()),
      props.doing,
      ANSWER_WORDS,
    )
    return
  }
  const landing = props.search.chooseItem(item, action)
  setSearchOpen(false)
  await openDestination(landing, props.places)
}

function onDismiss() {
  if (props.commands.open.value) props.commands.leaveStep()
  else props.search.setOpen(false)
}

function onBack() {
  if (props.commands.open.value && props.commands.goBack()) {
    props.search.setOpen(true)
  }
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
    :placeholder="field.placeholder"
    :crumb="field.crumb"
    :step="field.step"
    :opens-on="field.opensOn"
    :name="words.find"
    :action-words="actionWords"
    @update:model-value="onTyping"
    @choose="onChoose"
    @light="commands.previewItem"
    @back="onBack"
    @dismiss="onDismiss"
  >
    <template #icon="{ id }">
      <component :is="getRowIcon(id)" v-if="getRowIcon(id)" class="command-icon" />
    </template>
  </Palette>
</template>

<style scoped>
/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.command-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}
</style>
