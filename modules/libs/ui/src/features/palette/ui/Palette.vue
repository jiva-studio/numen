<script setup lang="ts">
/**
 * The palette: a field, and everything the words in it turned up, in groups.
 *
 * It takes groups of items and says which item was chosen and what was asked of
 * it. The keyboard stays in the field the whole time, what is lit is named to a
 * screen reader, and the window behind is out of reach for as long as the
 * palette stands.
 *
 * `data-palette` names each part of the panel: `ground`, `panel`, `crumb`,
 * `field`, `said`, `nothing`, `key` and `more`. The list under the field names
 * its own.
 */
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  useId,
  useTemplateRef,
  watch,
} from 'vue'
import PaletteKeyHints from './PaletteKeyHints.vue'
import { PaletteActions } from './palette-actions'
import { PaletteResults } from './palette-results'
import { ACTION_WORDS, type ActionWords } from '../lib/actions'
import {
  choosable,
  flatten,
  keptAt,
  ordered,
  stepTo,
  type PaletteGroup,
  type PaletteLit,
} from '../lib/item'
import { actionAt, commandKeyChord, keyed, opensActions } from '../lib/keys'
import { listId, optionId, placePalette } from '../lib/place'
import type { PaletteKeys } from '@/shared/ui/key-cap'

const props = withDefaults(
  defineProps<{
    /**
     * The groups, in the order they are offered. A group holding nothing is
     * drawn at the foot, whatever order it was offered in, and only while it
     * is working or has something to say in place of items.
     */
    groups?: readonly PaletteGroup[]
    /** Whether it is drawn at all. */
    open?: boolean
    /** The words standing in for what has not been typed. */
    placeholder?: string
    /**
     * Which step of a flow the field is on, in a few words, drawn beside it.
     * Nothing is drawn without it.
     */
    crumb?: string
    /**
     * Which step the caller is on. The identifier is opaque; changing it hands
     * the keyboard back to the field with what stands there selected.
     */
    step?: string
    /**
     * The item the keyboard stands on as a step opens: for a list of values,
     * the value in force. A list of no value, and a value with no item to
     * stand on, open on the first item there is.
     */
    opensOn?: string
    /** Where the keyboard goes back to once it closes. */
    from?: HTMLElement | null
    /** Where it is drawn. The end of the document by default. */
    to?: string | HTMLElement
    /** What it is announced as. */
    name?: string
    /** The words the action panel is drawn with. */
    actionWords?: ActionWords
    /** The keystroke that opens the action panel, as this machine reports it. */
    actionKey?: PaletteKeys
  }>(),
  {
    groups: () => [],
    open: false,
    placeholder: 'Search',
    crumb: '',
    step: '',
    opensOn: '',
    from: null,
    to: 'body',
    name: 'Palette',
    actionWords: () => ACTION_WORDS,
    actionKey: () => commandKeyChord(navigator.userAgent),
  },
)

const emit = defineEmits<{
  /**
   * An item was chosen, and what was asked of it. Both identifiers are the
   * caller's, handed back as given.
   */
  (event: 'choose', item: string, action: string): void
  /**
   * The item the keyboard is standing on, said whenever it moves. A list that
   * changes under the keyboard says this once, when it has settled.
   */
  (event: 'lit', item: PaletteLit): void
  /**
   * Backspace was pressed in an empty field. The caller keeps the steps and
   * decides what going back means.
   */
  (event: 'back'): void
  /** It asks to be put away. */
  (event: 'dismiss'): void
}>()

defineSlots<{
  /**
   * What is drawn before a row's words. The room for it is kept on every row
   * once the slot is filled, so the words line up down the list whether or not
   * each of them draws anything.
   */
  icon(props: { id: string }): unknown
  /** What is said while there is no group to draw. */
  silence(): unknown
}>()

const typed = defineModel<string>({ default: '' })

const uid = useId()

const field = useTemplateRef<HTMLInputElement>('field')

/** The list, for asking it to bring what is lit into sight. */
const results = useTemplateRef<InstanceType<typeof PaletteResults>>('results')

/** What is drawn, and in what order: a group holding nothing stands at the foot. */
const shown = computed(() => ordered(props.groups))

const places = computed(() => flatten(shown.value))
const placed = computed(() => placePalette(shown.value))

/**
 * What is read out of a search. A group that answered with rows is read out by
 * the row the keyboard lands on; a group that answered with none has no row to
 * land on, so what it says in place of one is read out instead. A group still
 * working has answered nothing yet and is left alone.
 */
const said = computed(() =>
  shown.value
    .filter((group) => !group.working && group.items.length === 0 && group.silence)
    .map((group) => group.silence)
    .join('. '),
)

/** The item the keyboard is on, by its identity. */
const held = ref('')

const here = computed(() => places.value.findIndex((place) => place.item.id === held.value))

/** The item the keyboard is on, and what it can be asked. */
const lit = computed(() => places.value[here.value]?.item)
const offered = computed(() => lit.value?.actions ?? [])

/** The actions a key reaches, which is what the foot of the palette says. */
const hinted = computed(() => keyed(lit.value))

const goTo = (at: number) => {
  held.value = places.value[at]?.item.id ?? ''
}

/** What is lit is brought into sight. Only a key does this. */
const reveal = () => results.value?.reveal(held.value)

/**
 * Answers arriving never move what is lit and never move the list. What they do
 * decide is the first item, when nothing is lit: a question was typed, and this
 * is its first answer.
 */
watch(
  places,
  (now) => {
    if (here.value >= 0) return
    goTo(stepTo(now, -1, 1))
  },
  { flush: 'post' },
)

/** A fresh question is a fresh list, and nothing in it is lit until it fills. */
watch(typed, () => {
  held.value = ''
})

/**
 * Where the keyboard is standing, for a caller that shows what it is standing
 * on. A pointer that has moved lights an item, so this is said for a pointer
 * crossing the list too.
 */
watch(held, (now) => emit('lit', now), { flush: 'post' })

/**
 * Where the pointer was. Only a pointer that has actually moved lights what it
 * is over: a list scrolling under a pointer standing still reports a move too.
 */
let stood = { x: -1, y: -1 }

const moved = (event: PointerEvent): boolean => {
  if (event.clientX === stood.x && event.clientY === stood.y) return false
  stood = { x: event.clientX, y: event.clientY }
  return true
}

const over = (at: number, event: PointerEvent) => {
  // The action panel is about the item that was lit when it opened, and stays
  // about it while a pointer crosses the list.
  if (panel.value) return
  if (!moved(event)) return
  // An item the keyboard steps over is one the pointer passes over.
  const item = places.value[at]?.item
  if (item && choosable(item)) goTo(at)
}

const choose = (at: number, second: boolean) => {
  const item = places.value[at]?.item
  const action = actionAt(item, second)
  if (!item || !action) return
  emit('choose', item.id, action)
}

/** Whether the action panel stands over the palette. */
const panel = ref(false)

/** An action chosen in the panel, on the item it was opened about. */
const ran = (action: string) => {
  const item = lit.value
  if (item) emit('choose', item.id, action)
}

/** An item that stops offering anything leaves the panel about nothing. */
watch(offered, (now) => {
  if (!now.length) panel.value = false
})

// The keyboard comes back to the field when the panel over it goes, and a
// palette that is going takes it somewhere else itself.
watch(panel, async (now) => {
  if (now || !props.open) return
  await nextTick()
  field.value?.focus()
})

/**
 * The keyboard, while the palette is open. Everything else reaches the field,
 * which is where a person is typing.
 */
const onKey = (event: KeyboardEvent) => {
  const step = (by: number, from = here.value) => {
    event.preventDefault()
    goTo(stepTo(places.value, from, by))
    void reveal()
  }
  // A chord that opens nothing is left to whoever else answers it.
  if (opensActions(event)) {
    if (!offered.value.length) return
    event.preventDefault()
    panel.value = true
  } else if (event.key === 'ArrowDown') step(1)
  else if (event.key === 'ArrowUp') step(-1)
  else if (event.key === 'Home') step(1, -1)
  else if (event.key === 'End') step(-1, 0)
  else if (event.key === 'Enter') {
    event.preventDefault()
    choose(here.value, event.shiftKey)
  } else if (event.key === 'Backspace' && typed.value === '') {
    event.preventDefault()
    emit('back')
  } else if (event.key === 'Escape') {
    event.preventDefault()
    emit('dismiss')
  }
  // The keyboard stays in the field for as long as the palette stands.
  else if (event.key === 'Tab') event.preventDefault()
}

/** A press anywhere but on the action panel puts the action panel away. */
const onPress = () => {
  panel.value = false
}

/** A press on the ground: the action panel goes, and the palette under it. */
const onGround = () => {
  if (panel.value) return void (panel.value = false)
  emit('dismiss')
}

/**
 * The keyboard put where the step wants it: on the value in force, else on
 * whatever it was standing on. Opening a step this way moves nothing, so a
 * caller that acts on what is lit acts on what is already so.
 */
const enter = async () => {
  panel.value = false
  goTo(keptAt(places.value, props.opensOn || held.value))
  await nextTick()
  field.value?.focus()
  field.value?.select()
  void reveal()
}

const leave = () => {
  panel.value = false
  held.value = ''
  const back = props.from
  if (back?.isConnected) back.focus()
}

watch(
  () => props.open,
  (now) => {
    if (now) void enter()
    else leave()
  },
)

/** A step of its own: its own question, and what stands in the field selected. */
watch(
  () => props.step,
  () => {
    if (props.open) void enter()
  },
)

onMounted(() => {
  if (props.open) void enter()
})

// A palette can go while it is still open, and the keyboard goes back with it.
onBeforeUnmount(() => {
  if (props.open) leave()
})
</script>

<template>
  <Teleport :to="to">
    <!-- The ground it stands on catches everything the panel does not, so a
         press anywhere else puts it away. -->
    <div
      v-if="open"
      class="palette numen font-sans text-base text-ink"
      data-palette="ground"
      @pointerdown.self="onGround"
    >
      <div
        class="palette__panel panel-numen relative flex min-h-0 flex-col"
        data-palette="panel"
        role="dialog"
        aria-modal="true"
        :aria-label="name"
        @keydown="onKey"
        @pointerdown="onPress"
      >
        <div class="palette__ask flex items-center gap-2">
          <!-- Which step the field is on. It says what the words typed here
               will mean. -->
          <span
            v-if="crumb"
            :id="`${uid}-crumb`"
            class="palette__crumb rounded-pill bg-bubble px-2 py-0.5 text-small"
            data-palette="crumb"
            >{{ crumb }}</span
          >
          <input
            ref="field"
            v-model="typed"
            class="palette__field w-full bg-transparent"
            data-palette="field"
            type="text"
            role="combobox"
            autocomplete="off"
            spellcheck="false"
            :placeholder="placeholder"
            :aria-label="name"
            :aria-describedby="crumb ? `${uid}-crumb` : undefined"
            :aria-expanded="placed.length !== 0"
            :aria-controls="listId(uid)"
            :aria-activedescendant="!panel && here >= 0 ? optionId(uid, here) : undefined"
          />
        </div>

        <!-- What a search came back with, where it came back with nothing. It
             stands here for as long as the palette does, so what lands in it
             is read out. -->
        <span class="sr-only" aria-live="polite" data-palette="said">{{ said }}</span>

        <PaletteResults
          v-if="placed.length"
          ref="results"
          :groups="placed"
          :here="here"
          :uid="uid"
          :name="name"
          @over="over"
          @choose="choose"
        >
          <template v-if="$slots.icon" #icon="{ id }">
            <slot name="icon" :id="id" />
          </template>
        </PaletteResults>

        <p
          v-else-if="$slots.silence"
          class="palette__nothing px-2 py-1.5 text-hushed"
          data-palette="nothing"
        >
          <slot name="silence" />
        </p>

        <!-- Everything the lit item offers, by name. It stands over the foot of
             the palette, and the list underneath stays where it was. -->
        <PaletteActions
          v-if="panel"
          v-model:open="panel"
          :offered="offered"
          :words="actionWords"
          @choose="ran"
        />

        <!-- What the item now lit can be asked. An item offering one action
             says one key, and the last word opens the rest. -->
        <PaletteKeyHints
          v-if="offered.length"
          :hinted="hinted"
          :action-key="actionKey"
          :name="actionWords.name"
        />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
/* Over the window, with what it covers showing through the panel and dimmed
   everywhere else. */
.palette {
  --lift: var(--numen-lift-palette);
  /* How far down the window it hangs, how wide it may be, and how much of the
     screen its list takes before it scrolls. */
  --drop: 12vh;
  --widest: 640px;
  --tallest: 50vh;

  position: fixed;
  inset: 0;
  z-index: var(--lift);
  display: grid;
  justify-items: center;
  align-content: start;
  padding: var(--drop) var(--numen-inset-wide);
  background: var(--numen-scrim);
}

.palette__panel {
  inline-size: 100%;
  max-inline-size: var(--widest);
  max-block-size: calc(var(--tallest) + 2 * var(--numen-action-size));
}

/* The clearance at the head of the row, which the step the field is on stands
   inside. */
.palette__ask {
  padding-inline-start: var(--numen-field-text-inset);
}

.palette__crumb {
  flex: none;
  max-inline-size: 45%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.palette__field {
  block-size: var(--numen-field-min);
  padding-inline-end: var(--numen-field-text-inset);
  border: 0;
  color: inherit;
  font: inherit;
  outline: none;
}

.palette__field::placeholder {
  color: var(--numen-edge-label);
}

/* The line under the field belongs to what stands beneath it. */
.palette__nothing {
  margin: 0;
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}

</style>
