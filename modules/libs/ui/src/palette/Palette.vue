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
 * typing. What is lit is named to a screen reader rather than focused.
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
  flatten,
  keptAt,
  ordered,
  placePalette,
  stepTo,
  type PaletteBand,
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
    /** Where the keyboard goes back to once it closes. */
    from?: HTMLElement | null
    /** Where it is drawn. The end of the document by default. */
    to?: string | HTMLElement
    /** What it is announced as. */
    name?: string
  }>(),
  {
    bands: () => [],
    open: false,
    placeholder: 'Search',
    from: null,
    to: 'body',
    name: 'Palette',
  },
)

const emit = defineEmits<{
  /**
   * An item was chosen, and what was asked of it. Both identifiers are the
   * caller's, handed back as given.
   */
  (event: 'choose', item: string, action: string): void
  /** It asks to be put away. */
  (event: 'dismiss'): void
}>()

defineSlots<{
  /** What is said while there is no band to draw. */
  silence(): unknown
}>()

const typed = defineModel<string>({ default: '' })

/**
 * The keys that reach an item's actions, in the order the actions are offered.
 * An item offering more actions than there are keys offers the rest to nothing.
 */
const KEYS = ['↵', '⇧↵'] as const

const uid = useId()
const optionName = (at: number): string => `${uid}-option-${at}`

const field = useTemplateRef<HTMLInputElement>('field')
const list = useTemplateRef<HTMLElement>('list')

/** What is drawn, and in what order: a band holding nothing stands at the foot. */
const shown = computed(() => ordered(props.bands))

const places = computed(() => flatten(shown.value))
const placed = computed(() => placePalette(shown.value))

/** The item the keyboard is on, by its identity rather than by where it sits. */
const held = ref('')

const here = computed(() => places.value.findIndex((place) => place.item.id === held.value))

/** What the item now lit can be asked, which is what the foot of the palette says. */
const offered = computed(() => places.value[here.value]?.item.actions ?? [])

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
 * A pointer that has actually moved lights what it is over. A list scrolling
 * under a pointer standing still reports a move too.
 */
let stood = { x: -1, y: -1 }

const over = (at: number, event: PointerEvent) => {
  if (event.clientX === stood.x && event.clientY === stood.y) return
  stood = { x: event.clientX, y: event.clientY }
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
 * The keyboard, while the palette is open. Everything else reaches the field,
 * which is where a person is typing.
 */
const onKey = (event: KeyboardEvent) => {
  const step = (by: number, from = here.value) => {
    event.preventDefault()
    goTo(stepTo(places.value, from, by))
    void reveal()
  }
  if (event.key === 'ArrowDown') step(1)
  else if (event.key === 'ArrowUp') step(-1)
  else if (event.key === 'Home') step(1, -1)
  else if (event.key === 'End') step(-1, 0)
  else if (event.key === 'Enter') {
    event.preventDefault()
    choose(here.value, event.shiftKey)
  } else if (event.key === 'Escape') {
    event.preventDefault()
    emit('dismiss')
  }
}

const enter = async () => {
  goTo(keptAt(places.value, held.value))
  await nextTick()
  field.value?.focus()
  field.value?.select()
  void reveal()
}

const leave = () => {
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
      @pointerdown.self="emit('dismiss')"
    >
      <div
        class="palette__panel flex min-h-0 flex-col rounded-panel border border-panel-rule bg-panel shadow-panel backdrop-blur-panel"
        role="dialog"
        :aria-label="name"
        @keydown="onKey"
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
          :aria-expanded="placed.length !== 0"
          :aria-controls="`${uid}-list`"
          :aria-activedescendant="here >= 0 ? optionName(here) : undefined"
        />

        <div
          v-if="placed.length"
          :id="`${uid}-list`"
          ref="list"
          class="palette__list min-h-0 flex-1"
          role="listbox"
          :aria-label="name"
        >
          <band
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
              class="palette__item flex flex-col rounded-node px-2 py-1.5"
              role="option"
              :aria-selected="drawn.at === here"
              :aria-disabled="drawn.item.disabled || undefined"
              :data-here="drawn.at === here || undefined"
              :data-off="drawn.item.disabled || undefined"
              @pointermove="over(drawn.at, $event)"
              @pointerdown.prevent
              @click="choose(drawn.at, $event.shiftKey)"
            >
              <span class="palette__name min-w-0">
                <span
                  v-for="(part, piece) in drawn.name"
                  :key="piece"
                  :data-hit="part.hit || undefined"
                  >{{ part.text }}</span
                >
              </span>

              <span v-if="drawn.detail.length" class="palette__detail min-w-0 text-small text-hushed">
                <span
                  v-for="(part, piece) in drawn.detail"
                  :key="piece"
                  :data-hit="part.hit || undefined"
                  >{{ part.text }}</span
                >
              </span>
            </div>

            <p v-if="!one.items.length" class="palette__silence px-2 py-1.5 text-hushed">
              {{ one.band.silence ?? 'Nothing' }}
            </p>
          </band>
        </div>

        <p v-else-if="$slots.silence" class="palette__nothing px-2 py-1.5 text-hushed">
          <slot name="silence" />
        </p>

        <!-- What the item now lit can be asked. An item offering one action
             says one key. -->
        <footer
          v-if="offered.length"
          class="palette__keys flex items-center gap-3 text-small text-hushed"
        >
          <span v-for="(action, at) in offered" :key="action.id" class="palette__key">
            <kbd>{{ KEYS[at] }}</kbd>
            {{ action.text }}
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

/* One row tall, with the clearance a field keeps from its own ends. */
.palette__field {
  block-size: var(--numen-field-min);
  padding-inline: var(--numen-field-text-inset);
  border: 0;
  border-block-end: var(--numen-stroke) solid var(--numen-panel-border);
  color: inherit;
  font: inherit;
  outline: none;
}

.palette__field::placeholder {
  color: var(--numen-edge-label);
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
  gap: 0.1rem;
  cursor: default;
  user-select: none;
}

.palette__item[data-here] {
  background: var(--numen-bubble-bg);
}

.palette__item[data-off] {
  color: var(--numen-edge-label);
}

/* One line each, then an ellipsis. A list is read down its leading edge. */
.palette__name,
.palette__detail {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Why the item is here. It sits under words that are being read, so it is a
   tint and not a colour. */
.palette__name [data-hit],
.palette__detail [data-hit] {
  border-radius: 2px;
  background: var(--numen-highlight);
  font-weight: 600;
}

.palette__silence,
.palette__nothing {
  margin: 0;
}

.palette__keys {
  padding: var(--numen-inset) var(--numen-inset-wide);
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}

.palette__key kbd {
  margin-inline-end: 0.35em;
  padding: 0.05em 0.35em;
  border: var(--numen-stroke) solid var(--numen-panel-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  font: inherit;
}
</style>
