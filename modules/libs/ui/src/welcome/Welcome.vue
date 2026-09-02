<script setup lang="ts">
/**
 * The screen a window opens on: the mark and the name, the ways in under the
 * keystrokes that reach them, the vaults this installation holds, and what
 * build this is in the corner.
 *
 * Both windows open on it. What each of them offers is its own; the screen
 * draws the rows it is given and decides nothing.
 */
import { FolderRoot } from '@lucide/vue'
import KeyCap from '../palette/KeyCap.vue'
import Mark from './Mark.vue'
import { vaultLetter } from './picking'
import type { Held, Offer, Way } from './welcome'

withDefaults(
  defineProps<{
    /**
     * What stands under the mark. Both windows wear the same glyph, and this is
     * what tells a person which of them they opened.
     */
    name?: string
    /** The ways in, above the list. A window offering none draws none. */
    ways?: readonly Way[]
    vaults: readonly Held[]
    /** What the list is called. */
    heading: string
    /** The row below the list, where a window offers one. */
    offer?: Offer | null
    version?: string
  }>(),
  { name: 'numen', ways: () => [], offer: null, version: '' },
)

defineEmits<{
  /** A way chosen, by the identifier the caller gave it. */
  (event: 'runs', id: string): void
  /** A vault on the list chosen. */
  (event: 'opens', id: string): void
  /** The row below the list pressed. */
  (event: 'offers'): void
}>()
</script>

<template>
  <div class="welcome">
    <div class="welcome__column">
      <div class="welcome__head">
        <Mark class="welcome__mark" />
        <h1 class="welcome__name">{{ name }}</h1>
      </div>

      <ul v-if="ways.length" class="welcome__ways">
        <li v-for="one in ways" :key="one.id">
          <button type="button" class="welcome__row" @click="$emit('runs', one.id)">
            <component :is="one.icon" v-if="one.icon" class="welcome__icon" />
            <span class="welcome__what">{{ one.text }}</span>
            <KeyCap v-if="one.keys" class="welcome__keys" :keys="one.keys" />
          </button>
        </li>
      </ul>

      <section class="welcome__vaults">
        <h2 class="welcome__heading">{{ heading }}</h2>
        <!-- The room the list will fill, while the window has no rows to give
             it and something to say about that. -->
        <div v-if="!vaults.length && $slots.waiting" class="welcome__waiting">
          <slot name="waiting" />
        </div>
        <ul v-else class="welcome__list">
          <li v-for="(one, at) in vaults" :key="one.id">
            <button
              type="button"
              class="welcome__row welcome__row--vault"
              :disabled="one.waiting"
              @click="$emit('opens', one.id)"
            >
              <FolderRoot class="welcome__icon" />
              <span class="welcome__named">
                <span class="welcome__what">{{ one.name }}</span>
                <!-- The whole path is on the element, for one too long to be drawn. -->
                <span class="welcome__aside" :title="one.path">{{ one.path }}</span>
              </span>
              <!-- What the window has to say about this one, drawn at the far
                   end of its row. What that is belongs to the window. -->
              <slot name="vault" :vault="one" />
              <span v-if="one.detail" class="welcome__state">{{ one.detail }}</span>
              <!-- The letter it is opened by, at the end of the row the ways in
                   carry their keystrokes at. Past the alphabet a vault is opened
                   with the hand and carries none, and a row still waiting is
                   drawn without the letter it will be opened by. -->
              <KeyCap
                v-if="vaultLetter(at) && !one.waiting"
                class="welcome__keys"
                :keys="{ marks: [], letter: vaultLetter(at) }"
              />
            </button>
          </li>
        </ul>
        <button v-if="offer" type="button" class="welcome__row" @click="$emit('offers')">
          <component :is="offer.icon" v-if="offer.icon" class="welcome__icon" />
          <span class="welcome__named">
            <span class="welcome__what">{{ offer.text }}</span>
            <span v-if="offer.detail" class="welcome__aside">{{ offer.detail }}</span>
          </span>
          <KeyCap v-if="offer.keys" class="welcome__keys" :keys="offer.keys" />
        </button>
      </section>
    </div>

    <span class="welcome__version">{{ version }}</span>
  </div>
</template>

<style scoped>
/* One column in the middle of the window, held to the width of a short line so
   the rows read as a list and not as a page. */
.welcome {
  /* How tall the glyph stands over the name. */
  --mark: 5.4rem;

  position: relative;
  display: flex;
  block-size: 100%;
  overflow: auto;
  container-type: inline-size;
  padding: var(--numen-gutter);
  color: var(--numen-node-fg);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
}

/* What build this is, in the far corner, where the column never reaches. */
.welcome__version {
  position: absolute;
  inset-block-end: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}

/* The column sits in the middle of whatever room there is. It is centred by its
   own margins, so a column taller than the room keeps its head inside it and
   the whole of it is scrolled to. */
.welcome__column {
  display: flex;
  flex-direction: column;
  gap: 1.4rem;
  margin: auto;
  inline-size: 100%;
  max-inline-size: 22rem;
}

.welcome__head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.welcome__mark {
  inline-size: auto;
  block-size: var(--mark);
}

/* The name is set in the letters the mark is drawn in, which are a serif's.
   Nothing else in the window is, so the family is this screen's own. */
.welcome__name {
  margin: 0;
  /* The glyph above is an italic and leans right, so the name sits a little
     left of the middle to read as under it. */
  margin-inline-end: 0.22em;
  font-family: ui-serif, Georgia, 'Times New Roman', serif;
  font-size: var(--numen-display-size);
  font-weight: 400;
  letter-spacing: 0.06em;
}

.welcome__ways,
.welcome__list {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.welcome__row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  inline-size: 100%;
  padding: 0.3rem 0.6rem;
  border: 0;
  border-radius: var(--numen-radius);
  background: none;
  color: inherit;
  font: inherit;
  text-align: start;
  cursor: pointer;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.welcome__icon {
  flex: none;
  inline-size: 1rem;
  block-size: 1rem;
  stroke-width: 1.75;
  opacity: 0.75;
}

/* A row is its name over what is said about it, beside the one icon. */
.welcome__named {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 0.05rem;
  min-inline-size: 0;
}

.welcome__row:hover:not(:disabled) {
  background: var(--numen-bubble-bg);
}

/* A row whose answer is still on its way stands as it will stand, and does not
   answer to a hand. */
.welcome__row:disabled {
  cursor: default;
}

.welcome__row:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: 1px;
}

/* A name is as long as a person makes it, and a long one ends in an ellipsis.
   It takes the room a row leaves it; inside a name over its second line the
   column above holds the room, and the name takes the height of its own line. */
.welcome__what {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.welcome__row > .welcome__what {
  flex: 1;
}

/* One line, then an ellipsis: a path is as long as the machine makes it. */
.welcome__aside {
  overflow: hidden;
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* What is true of one row of the list and not of the ones beside it, said at
   the far end of it. */
.welcome__state {
  flex: none;
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}

/* Where the rows will stand, so what is said while they are on their way is
   said in the middle of the room they will take. */
.welcome__waiting {
  display: flex;
  align-items: center;
  justify-content: center;
  min-block-size: 4rem;
}

/* Narrow, the row keeps its name and gives up the keystroke drawn on it: the
   row is pressed by hand, and the key still works. Narrower, it gives up what
   is said under the name; a path is on the row itself, for pointing at. */
@container (max-width: 24rem) {
  .welcome__keys {
    display: none;
  }
}

@container (max-width: 18rem) {
  .welcome__aside {
    display: none;
  }
}

.welcome__heading {
  margin: 0 0 0.25rem;
  padding-inline: 0.6rem;
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
  font-weight: inherit;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}
</style>
