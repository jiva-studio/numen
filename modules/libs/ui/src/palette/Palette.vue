<script setup lang="ts">
/**
 * The palette: a field, and everything the words in it turned up, in bands.
 *
 * It stands over the whole window and is opened by a keystroke, so it is drawn
 * at the end of the document and belongs to nothing on screen. It takes bands
 * of items and says which item was chosen and what was asked of it; what the
 * bands are, what an item addresses and what choosing one does are the
 * caller's.
 *
 * The keyboard stays in the field the whole time, because a person is still
 * typing. What is lit is named to a screen reader rather than focused. The one
 * place the keyboard leaves the field is the action panel, which is a field of
 * its own over a list of its own.
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
import Waiting from '../waiting/Waiting.vue'
import {
  actionAt,
  choosable,
  commandKeyWord,
  flatten,
  keptAt,
  keptOn,
  keyed,
  opensActions,
  ordered,
  placeActions,
  placePalette,
  stepIn,
  stepTo,
  type PaletteBand,
  type PaletteLit,
} from './model'

const props = withDefaults(
  defineProps<{
    /**
     * The bands, in the order they are offered. A band holding nothing is
     * drawn at the foot, whatever order it was offered in.
     */
    bands?: readonly PaletteBand[]
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
    /** What the action panel is announced as, and what the key to it is called. */
    actionsName?: string
    /** The words standing in for what has not been typed in the panel's field. */
    actionsPlaceholder?: string
    /** What the panel says when the words in its field leave no action. */
    actionsSilence?: string
  }>(),
  {
    bands: () => [],
    open: false,
    placeholder: 'Search',
    crumb: '',
    step: '',
    opensOn: '',
    from: null,
    to: 'body',
    name: 'Palette',
    actionsName: 'Actions',
    actionsPlaceholder: 'Search actions',
    actionsSilence: 'Nothing by that name',
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
  /** What is said while there is no band to draw. */
  silence(): unknown
}>()

const typed = defineModel<string>({ default: '' })

/** What the key that opens the action panel is written as on this keyboard. */
const command = commandKeyWord(navigator.userAgent)

const uid = useId()
const optionName = (at: number): string => `${uid}-option-${at}`
const actionName = (at: number): string => `${uid}-action-${at}`

const field = useTemplateRef<HTMLInputElement>('field')
const list = useTemplateRef<HTMLElement>('list')
const sheet = useTemplateRef<HTMLElement>('sheet')
const hunt = useTemplateRef<HTMLInputElement>('hunt')

/** What is drawn, and in what order: a band holding nothing stands at the foot. */
const shown = computed(() => ordered(props.bands))

const places = computed(() => flatten(shown.value))
const placed = computed(() => placePalette(shown.value))

/** The item the keyboard is on, by its identity rather than by where it sits. */
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
const reveal = async () => {
  await nextTick()
  list.value?.querySelector<HTMLElement>('[data-here]')?.scrollIntoView?.({ block: 'nearest' })
}

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

/**
 * The action panel: everything the lit item offers, by name, in a list of its
 * own with a field of its own. Every action is a keystroke and a name away.
 */
const panel = ref(false)

/** What is typed in the panel's field, which narrows the actions and nothing else. */
const hunted = ref('')

/** The action the panel is on, by its identity rather than by where it sits. */
const chosen = ref('')

const actions = computed(() => placeActions(offered.value, hunted.value))

/** Which row that action stands on, counted over the actions the words left. */
const actionHere = computed(() => actions.value.findIndex((one) => one.action.id === chosen.value))

const goToAction = (to: number) => {
  chosen.value = actions.value[to]?.action.id ?? ''
}

const overAction = (to: number, event: PointerEvent) => {
  if (moved(event)) goToAction(to)
}

const revealAction = async () => {
  await nextTick()
  sheet.value?.querySelector<HTMLElement>('[data-here]')?.scrollIntoView?.({ block: 'nearest' })
}

const raise = async () => {
  if (!offered.value.length) return
  panel.value = true
  hunted.value = ''
  goToAction(keptOn(actions.value, ''))
  await nextTick()
  hunt.value?.focus()
}

const shut = async () => {
  if (!panel.value) return
  panel.value = false
  chosen.value = ''
  await nextTick()
  field.value?.focus()
}

const run = (to: number) => {
  const chosen = actions.value[to]
  const item = lit.value
  if (!chosen || !item) return
  emit('choose', item.id, chosen.action.id)
  void shut()
}

/** A fresh list keeps the action the panel was on, wherever the words put it. */
watch(actions, (now) => {
  if (panel.value) goToAction(keptOn(now, chosen.value))
})

/** An item that stops offering anything leaves the panel about nothing. */
watch(offered, (now) => {
  if (!now.length) void shut()
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
    void raise()
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

/**
 * The keyboard, while the action panel is open. Nothing of it reaches the
 * palette underneath: the panel is where a person is now typing.
 */
const onActionKey = (event: KeyboardEvent) => {
  const step = (by: number, from = actionHere.value) => {
    event.preventDefault()
    goToAction(stepIn(actions.value.length, from, by))
    void revealAction()
  }
  if (opensActions(event) || event.key === 'Escape') {
    event.preventDefault()
    void shut()
  } else if (event.key === 'ArrowDown') step(1)
  else if (event.key === 'ArrowUp') step(-1)
  else if (event.key === 'Home') step(1, -1)
  else if (event.key === 'End') step(-1, 0)
  else if (event.key === 'Enter') {
    event.preventDefault()
    run(actionHere.value)
  }
  // The keyboard stays in the panel's field for as long as the panel stands.
  else if (event.key === 'Tab') event.preventDefault()
}

/** A press anywhere but on the action panel puts the action panel away. */
const onPress = (event: PointerEvent) => {
  const target = event.target
  if (target instanceof Node && sheet.value?.contains(target)) return
  void shut()
}

/** A press on the ground: the action panel goes, and the palette under it. */
const onGround = () => {
  if (panel.value) return void shut()
  emit('dismiss')
}

/**
 * The keyboard put where the step wants it: on the value in force, else on
 * whatever it was standing on. Opening a step this way moves nothing, so a
 * caller that acts on what is lit acts on what is already so.
 */
const enter = async () => {
  panel.value = false
  chosen.value = ''
  goTo(keptAt(places.value, props.opensOn || held.value))
  await nextTick()
  field.value?.focus()
  field.value?.select()
  void reveal()
}

const leave = () => {
  panel.value = false
  chosen.value = ''
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
      @pointerdown.self="onGround"
    >
      <div
        class="palette__panel relative flex min-h-0 flex-col rounded-panel border border-panel-rule bg-panel shadow-panel backdrop-blur-panel"
        role="dialog"
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
            >{{ crumb }}</span
          >
          <input
            ref="field"
            v-model="typed"
            class="palette__field w-full bg-transparent"
            type="text"
            role="combobox"
            autocomplete="off"
            spellcheck="false"
            :placeholder="placeholder"
            :aria-label="name"
            :aria-describedby="crumb ? `${uid}-crumb` : undefined"
            :aria-expanded="placed.length !== 0"
            :aria-controls="`${uid}-list`"
            :aria-activedescendant="!panel && here >= 0 ? optionName(here) : undefined"
          />
        </div>

        <div
          v-if="placed.length"
          :id="`${uid}-list`"
          ref="list"
          class="palette__list min-h-0 flex-1"
          role="listbox"
          :aria-label="name"
        >
          <section
            v-for="one in placed"
            :key="one.band.id"
            class="palette__band"
            role="group"
            :aria-labelledby="`${uid}-band-${one.band.id}`"
            :aria-busy="one.band.working || undefined"
          >
            <p
              :id="`${uid}-band-${one.band.id}`"
              class="palette__title flex items-center gap-1.5 text-small text-hushed"
            >
              <span>{{ one.band.title }}</span>
              <!-- More of this band is on its way. -->
              <Waiting v-if="one.band.working" />
            </p>
            <div
              v-for="drawn in one.items"
              :id="optionName(drawn.at)"
              :key="drawn.item.id"
              class="palette__item flex items-center gap-2 rounded-node px-2 py-1.5"
              role="option"
              :aria-selected="drawn.at === here"
              :aria-disabled="drawn.item.disabled || undefined"
              :data-here="drawn.at === here || undefined"
              :data-off="drawn.item.disabled || undefined"
              @pointermove="over(drawn.at, $event)"
              @pointerdown.prevent
              @click="choose(drawn.at, $event.shiftKey)"
            >
              <span class="palette__lines flex min-w-0 flex-1 flex-col">
                <span class="palette__name min-w-0">
                  <span
                    v-for="(part, piece) in drawn.name"
                    :key="piece"
                    :data-hit="part.hit || undefined"
                    >{{ part.text }}</span
                  >
                </span>

                <span
                  v-if="drawn.detail.length"
                  class="palette__detail min-w-0 text-small text-hushed"
                >
                  <span
                    v-for="(part, piece) in drawn.detail"
                    :key="piece"
                    :data-hit="part.hit || undefined"
                    >{{ part.text }}</span
                  >
                </span>
              </span>

              <!-- What reaches this item away from the palette. -->
              <kbd v-if="drawn.item.keys" class="palette__hint text-small text-hushed">{{
                drawn.item.keys
              }}</kbd>
            </div>

            <p v-if="!one.items.length" class="palette__silence px-2 py-1.5 text-hushed">
              {{ one.band.silence ?? 'Nothing' }}
            </p>
          </section>
        </div>

        <p v-else-if="$slots.silence" class="palette__nothing px-2 py-1.5 text-hushed">
          <slot name="silence" />
        </p>

        <!-- Everything the lit item offers, by name. It stands over the foot of
             the palette, and the list underneath stays where it was. -->
        <div
          v-if="panel"
          ref="sheet"
          class="palette__actions flex flex-col rounded-panel border border-panel-rule bg-panel shadow-panel backdrop-blur-panel"
          role="dialog"
          :aria-label="actionsName"
          @keydown.stop="onActionKey"
        >
          <div
            :id="`${uid}-actions`"
            class="palette__deeds min-h-0 flex-1"
            role="listbox"
            :aria-label="actionsName"
          >
            <div
              v-for="deed in actions"
              :id="actionName(deed.at)"
              :key="deed.action.id"
              class="palette__deed flex items-center gap-3 rounded-node px-2 py-1.5"
              role="option"
              :aria-selected="deed.at === actionHere"
              :data-here="deed.at === actionHere || undefined"
              @pointermove="overAction(deed.at, $event)"
              @pointerdown.prevent
              @click="run(deed.at)"
            >
              <span class="palette__deed-name min-w-0 flex-1">
                <span
                  v-for="(part, piece) in deed.name"
                  :key="piece"
                  :data-hit="part.hit || undefined"
                  >{{ part.text }}</span
                >
              </span>
              <kbd v-if="deed.key" class="palette__hint text-small text-hushed">{{ deed.key }}</kbd>
            </div>
          </div>

          <p v-if="!actions.length" class="palette__deed-silence px-2 py-1.5 text-hushed">
            {{ actionsSilence }}
          </p>

          <input
            ref="hunt"
            v-model="hunted"
            class="palette__hunt w-full bg-transparent"
            type="text"
            role="combobox"
            autocomplete="off"
            spellcheck="false"
            :placeholder="actionsPlaceholder"
            :aria-label="actionsPlaceholder"
            :aria-expanded="actions.length !== 0"
            :aria-controls="`${uid}-actions`"
            :aria-activedescendant="actionHere >= 0 ? actionName(actionHere) : undefined"
          />
        </div>

        <!-- What the item now lit can be asked. An item offering one action
             says one key, and the last word opens the rest. -->
        <footer
          v-if="offered.length"
          class="palette__keys flex items-center gap-3 text-small text-hushed"
        >
          <span v-for="one in hinted" :key="one.action.id" class="palette__key">
            <kbd>{{ one.key }}</kbd>
            {{ one.action.text }}
          </span>
          <span class="palette__more ml-auto">
            <kbd>{{ command }}</kbd>
            {{ actionsName }}
          </span>
        </footer>
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

.palette__field,
.palette__hunt {
  block-size: var(--numen-field-min);
  padding-inline-end: var(--numen-field-text-inset);
  border: 0;
  color: inherit;
  font: inherit;
  outline: none;
}

.palette__hunt {
  padding-inline-start: var(--numen-field-text-inset);
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}

.palette__field::placeholder,
.palette__hunt::placeholder {
  color: var(--numen-edge-label);
}

/* The line under the field belongs to what stands beneath it. */
.palette__list,
.palette__nothing {
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}

.palette__list {
  max-block-size: var(--tallest);
  padding: var(--numen-field-padding);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.palette__band + .palette__band {
  margin-block-start: var(--numen-panel-gap);
}

/* The band's name is small print over what it names. */
.palette__title {
  margin: 0;
  padding: 0.15rem 0.5rem;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.palette__item {
  cursor: default;
  user-select: none;
}

.palette__lines {
  gap: 0.1rem;
}

.palette__item[data-here] {
  background: var(--numen-bubble-bg);
}

.palette__item[data-off] {
  color: var(--numen-edge-label);
}

/* One line each, then an ellipsis. A list is read down its leading edge. */
.palette__name,
.palette__detail,
.palette__deed-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Why the item is here. It sits under words that are being read, so it is a
   tint and not a colour. */
.palette__name [data-hit],
.palette__detail [data-hit],
.palette__deed-name [data-hit] {
  border-radius: 2px;
  background: var(--numen-highlight);
  font-weight: 600;
}

.palette__silence,
.palette__nothing,
.palette__deed-silence {
  margin: 0;
}

.palette__keys {
  padding: var(--numen-inset) var(--numen-inset-wide);
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}

.palette__key kbd,
.palette__more kbd,
.palette__hint {
  margin-inline-end: 0.35em;
  padding: 0.05em 0.35em;
  border: var(--numen-stroke) solid var(--numen-panel-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  font: inherit;
}

/* A key written on a row is the last thing on it, and is read after the name. */
.palette__hint {
  flex: none;
  margin-inline-end: 0;
}

/* The actions stand over the foot of the palette, at the corner the keys are
   read from. */
.palette__actions {
  /* How wide the panel is and how much of the palette its list takes. */
  --panel-width: 280px;
  --panel-tallest: 240px;

  position: absolute;
  inset-block-end: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  inline-size: var(--panel-width);
  /* Never past the corners of the palette it stands over. */
  max-inline-size: calc(100% - 2 * var(--numen-inset));
  max-block-size: calc(100% - 2 * var(--numen-inset));
}

.palette__deeds {
  max-block-size: var(--panel-tallest);
  padding: var(--numen-field-padding);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.palette__deed {
  cursor: default;
  user-select: none;
}

.palette__deed[data-here] {
  background: var(--numen-bubble-bg);
}
</style>
