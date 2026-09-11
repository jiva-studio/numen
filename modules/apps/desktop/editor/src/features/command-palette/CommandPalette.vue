<script setup lang="ts">
/**
 * The one field the window is asked things through: the commands over what is
 * in front, and the search under them.
 *
 * The two share a field, and which of them is open decides what is drawn in it
 * and what a keystroke in it means. Two chords put each up and take it down.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { Palette, type ActionWords } from '@numen/ui'
import { asksCommands, type CommandTarget } from './target'
import { creates, MAKING, offering } from './offers'
import type { Commands } from './palette'
import type { CommandDeps } from './deps'
import { does } from './handlers'
import type { SearchState } from './search'
import { iconFor, iconOfNote, iconOfSource } from '../../shared/icons'
import { chorded } from './chords'
import { lands, type DestinationDeps } from './destination'
import { WORDS as words } from '../../shared/words'

// --- Props & Emits ---
const props = defineProps<{
  commands: Commands
  search: SearchState
  doing: CommandDeps
  /** What the commands are over, as the window stands now. */
  where: () => CommandTarget
  /** Where the window is taken by what the search turns up. */
  places: DestinationDeps
}>()

// --- State ---
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
        groups: offering(props.search.groups.value, props.search.typed.value, words, props.where()),
        crumb: '',
        step: '',
        opensOn: '',
        placeholder: words.find,
      },
)

// --- Handlers ---
function onTyping(text: string) {
  if (props.commands.open.value) return void props.commands.typing(text)
  if (!asksCommands(props.search.typed.value, text)) return void props.search.typing(text)
  const setSearchOpen = props.search.setOpen ?? props.search.shows
  const setCommandsOpen = props.commands.setOpen ?? props.commands.shows
  setSearchOpen(false)
  setCommandsOpen(true)
}

async function onChoose(item: string, action: string) {
  const setSearchOpen = props.search.setOpen ?? props.search.shows
  const setCommandsOpen = props.commands.setOpen ?? props.commands.shows
  if (props.commands.open.value) {
    const invocation = props.commands.chose(item, action)
    if (!invocation) return
    setCommandsOpen(false)
    await does(invocation, props.doing, words)
    return
  }
  if (item === MAKING) {
    const name = props.search.typed.value.trim()
    setSearchOpen(false)
    await does(creates(action, name, props.where()), props.doing, words)
    return
  }
  const landing = props.search.chose(item, action)
  setSearchOpen(false)
  await lands(landing, props.places)
}

function onDismiss() {
  if (props.commands.open.value) props.commands.leaves()
  else (props.search.setOpen ?? props.search.shows)(false)
}

function onBack() {
  if (props.commands.open.value && props.commands.backs()) {
    ;(props.search.setOpen ?? props.search.shows)(true)
  }
}

function onKeyDown(event: KeyboardEvent) {
  if (event.defaultPrevented || !chorded(event) || event.shiftKey) return
  const key = event.key.toLowerCase()
  const setSearchOpen = props.search.setOpen ?? props.search.shows
  const setCommandsOpen = props.commands.setOpen ?? props.commands.shows
  if (key === 'k') {
    event.preventDefault()
    setCommandsOpen(false)
    setSearchOpen(!props.search.open.value)
    return
  }
  if (key === 'p') {
    event.preventDefault()
    setSearchOpen(false)
    setCommandsOpen(!props.commands.open.value)
  }
}

onMounted(() => globalThis.addEventListener('keydown', onKeyDown))
onUnmounted(() => globalThis.removeEventListener('keydown', onKeyDown))

// --- Helpers ---
function rowIcon(id: string) {
  if (props.commands.open.value) {
    const picked = props.commands.typeOf(id)
    return picked ? iconOfNote(picked) : iconFor(id)
  }
  if (id === MAKING) return iconFor('note')
  const type = props.search.typeOf(id)
  if (type) return iconOfNote(type)
  const kind = props.search.kindOf(id)
  return kind ? iconOfSource(kind) : null
}
</script>

<template>
  <Palette
    :model-value="field.typed"
    :groups="field.groups"
    :open="field.open"
    :placeholder="field.placeholder"
    :crumb="field.crumb"
    :step="field.step"
    :opens-on="field.opensOn"
    :name="words.find"
    :action-words="actionWords"
    @update:model-value="onTyping"
    @choose="onChoose"
    @lit="commands.lights"
    @back="onBack"
    @dismiss="onDismiss"
  >
    <template #icon="{ id }">
      <component :is="rowIcon(id)" v-if="rowIcon(id)" class="command-icon" />
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
