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
import { isCommandsTyped } from '../lib/commands'
import type { CommandDeps } from '../model/deps'
import type { CommandTarget } from '../types'
import { appendCreateOffer, createNoteInvocation, MAKING } from '../lib/offers'
import type { Commands } from '../model/palette'
import { runInvocation } from '../model/handlers'
import type { SearchState } from '../model/search'
import { iconOfSource } from '@/entities/file'
import { iconOfNote } from '@/entities/note'
import { iconFor } from '@/shared/icons'
import { isChord } from '../lib/chords'
import { openDestination, type DestinationDeps } from '../model/destination'
import { ANSWER_WORDS } from '../words'
import { WORDS as words } from '@/shared/words'

// --- Props & Emits ---
const props = defineProps<{
  commands: Commands
  search: SearchState
  doing: CommandDeps
  /** What the commands are over, as the window stands now. */
  getTarget: () => CommandTarget
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
        groups: appendCreateOffer(
          props.search.groups.value,
          props.search.typed.value,
          words,
          props.getTarget(),
        ),
        crumb: '',
        step: '',
        opensOn: '',
        placeholder: words.find,
      },
)

// --- Handlers ---
function onTyping(text: string) {
  if (props.commands.open.value) return void props.commands.setTyped(text)
  if (!isCommandsTyped(props.search.typed.value, text)) return void props.search.setTyped(text)
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
  if (item === MAKING) {
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

function onKeyDown(event: KeyboardEvent) {
  if (event.defaultPrevented || !isChord(event) || event.shiftKey) return
  const key = event.key.toLowerCase()
  const setSearchOpen = props.search.setOpen
  const setCommandsOpen = props.commands.setOpen
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
    @light="commands.previewItem"
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
