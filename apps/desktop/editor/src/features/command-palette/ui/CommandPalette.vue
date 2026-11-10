<script setup lang="ts">
/**
 * The single input field of the window: the commands for the current target,
 * and the search.
 *
 * The two share a field, and which one is active determines what is drawn in it
 * and what a keystroke in it means. The keys of the window open and close each.
 */
import { computed } from 'vue'
import { Search } from '@lucide/vue'
import { Palette, type ActionWords } from '@numen/workbench'
import { isCommandPrefixTyped } from '../lib/commands/commands'
import type { CommandDeps } from '../model/handlers/deps'
import type { CommandTarget } from '@/entities/command'
import type { Commands } from '../model/palette/palette'
import { runInvocation } from '../model/handlers/handlers'
import type { ResultChoice, SearchState } from '../model/search/search'
import { findRowIcon, type FileIcons } from '../model/rowIcons'
import { ANSWER_WORDS } from '../words'
import { WORDS as words } from '@/shared/words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  commands: Commands
  search: SearchState
  doing: CommandDeps
  getTarget: () => CommandTarget
  places: FileIcons
  runChoice: (choice: ResultChoice) => void
}>()

/* --------------------------------- State ---------------------------------- */
const actionWords: ActionWords = {
  name: words.actions,
  placeholder: words.findAction,
  silence: words.noAction,
}

/** What the palette draws: the commands while they are open, the search under. */
const field = computed(() =>
  props.commands.open.value
    ? {
        open: true,
        isBusy: false,
        typed: props.commands.typed.value,
        groups: props.commands.groups.value,
        crumb: props.commands.crumb.value,
        step: props.commands.step.value,
        opensOn: props.commands.opensOn.value,
        placeholder: props.commands.placeholder.value,
      }
    : {
        open: props.search.open.value,
        isBusy: props.search.status.isBusy.value,
        typed: props.search.typed.value,
        groups: props.search.groups.value,
        crumb: '',
        step: '',
        opensOn: '',
        placeholder: words.find,
      },
)

/* -------------------------------- Handlers -------------------------------- */
async function onTyping(text: string) {
  if (props.commands.open.value) return await props.commands.setTyped(text)
  if (!isCommandPrefixTyped(props.search.typed.value, text))
    return await props.search.setTyped(text)
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
  const choice = props.search.chooseResult(item, action)
  setSearchOpen(false)
  if (choice) props.runChoice(choice)
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
    :is-busy="field.isBusy"
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
    <template v-if="!commands.open.value && search.status.isNothingFound.value" #silence>
      <span class="nothing-found">
        <Search class="nothing-found__icon" data-palette="nothing-icon" aria-hidden="true" />
        <span class="nothing-found__text" data-palette="nothing-text">{{
          words.nothingFound(search.typed.value.trim())
        }}</span>
      </span>
    </template>
  </Palette>
</template>

<style scoped>
.nothing-found {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: center;
  padding-block: 1.5rem;
  text-align: center;
}

.nothing-found__icon {
  flex: none;
  inline-size: 1.5rem;
  block-size: 1.5rem;
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
