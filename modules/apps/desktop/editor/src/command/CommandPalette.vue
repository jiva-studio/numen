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
import {
  asksCommands,
  creates,
  MAKING,
  offering,
  type Commands,
  type CommandTarget,
} from './commands'
import { does, type CommandDeps } from './handlers'
import type { SearchState } from './search'
import { iconFor, iconOfNote, iconOfSource } from '../icons'
import { chorded } from './chords'
import { lands, type DestinationDeps } from './destination'
import { WORDS as words } from '../words'

const props = defineProps<{
  commands: Commands
  search: SearchState
  doing: CommandDeps
  /** What the commands are over, as the window stands now. */
  where: () => CommandTarget
  /** Where the window is taken by what the search turns up. */
  places: DestinationDeps
}>()

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

/**
 * What one row of the palette is drawn as: a command by its own mark, a name or
 * a passage by the kind of note it stands in, and the note a search did not
 * find by the mark of making one. A passage out of a book or a recording stands
 * in no note and is drawn as the source it was read out of.
 */
const rowIcon = (id: string) => {
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

/**
 * Something typed in the field. The one character that means the commands is
 * the one typed into a field holding nothing.
 */
const typing = (text: string) => {
  if (props.commands.open.value) return void props.commands.typing(text)
  if (!asksCommands(props.search.typed.value, text)) return void props.search.typing(text)
  props.search.shows(false)
  props.commands.shows(true)
}

/** Something chosen, and the window taken there or the command carried out. */
const went = async (item: string, action: string) => {
  if (props.commands.open.value) {
    const deed = props.commands.chose(item, action)
    if (!deed) return
    props.commands.shows(false)
    await does(deed, props.doing, words)
    return
  }
  if (item === MAKING) {
    const name = props.search.typed.value.trim()
    props.search.shows(false)
    await does(creates(action, name, props.where()), props.doing, words)
    return
  }
  const landing = props.search.chose(item, action)
  props.search.shows(false)
  await lands(landing, props.places)
}

/** Escape: a step of a command goes, and the palette itself at the last of them. */
const dismissed = () =>
  props.commands.open.value ? props.commands.leaves() : props.search.shows(false)

/** Backspace in an empty field: the step goes, and the first hands back the search. */
const back = () => {
  if (props.commands.open.value && props.commands.backs()) props.search.shows(true)
}

/**
 * The two chords that put the field up and take it down. Each is that letter
 * alone: the same letter with Shift is a chord of its own and goes to whoever
 * the table of keystrokes gives it to.
 */
const asked = (event: KeyboardEvent) => {
  if (event.defaultPrevented || !chorded(event) || event.shiftKey) return
  const key = event.key.toLowerCase()
  if (key === 'k') {
    event.preventDefault()
    props.commands.shows(false)
    props.search.shows(!props.search.open.value)
    return
  }
  if (key === 'p') {
    event.preventDefault()
    props.search.shows(false)
    props.commands.shows(!props.commands.open.value)
  }
}

/** What the palette's action panel is drawn with. */
const actionWords: ActionWords = {
  name: words.actions,
  placeholder: words.findAction,
  silence: words.noAction,
}

onMounted(() => globalThis.addEventListener('keydown', asked))
onUnmounted(() => globalThis.removeEventListener('keydown', asked))
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
    @update:model-value="typing"
    @choose="went"
    @lit="commands.lights"
    @back="back"
    @dismiss="dismissed"
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
