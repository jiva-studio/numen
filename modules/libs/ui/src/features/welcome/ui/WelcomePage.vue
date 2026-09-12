<script setup lang="ts">
/**
 * The screen a window opens on: the mark and the name, the ways in under the
 * keystrokes that reach them, the vaults this installation holds, and what
 * build this is in the corner.
 *
 * Both windows open on it. What each of them offers is its own; the screen
 * draws the rows it is given and decides nothing.
 */
import { Glyph } from './glyph'
import { WelcomeRow } from './welcome-row'
import { WelcomeVaults } from './welcome-vaults'
import type { Offer, VaultRow, WelcomeAction } from '../lib/welcome'

withDefaults(
  defineProps<{
    /**
     * What stands under the mark. Both windows wear the same glyph, and this is
     * what tells a person which of them they opened.
     */
    name?: string
    /** The ways in, above the list. A window offering none draws none. */
    ways?: readonly WelcomeAction[]
    vaults: readonly VaultRow[]
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

defineSlots<{
  /** What stands where the list would be while there is no list yet. */
  waiting?(): unknown
  /** How a vault on the list is drawn. */
  vault?(props: { vault: VaultRow }): unknown
}>()
</script>

<template>
  <div class="welcome-page">
    <div class="welcome-page__column">
      <div class="welcome-page__lead">
        <div class="welcome-page__head">
          <Glyph class="welcome-page__glyph" />
          <h1 class="welcome-page__name">{{ name }}</h1>
        </div>

        <ul v-if="ways.length" class="welcome-page__ways">
          <li v-for="one in ways" :key="one.id">
            <WelcomeRow
              :icon="one.icon"
              :text="one.text"
              :keys="one.keys"
              @click="$emit('runs', one.id)"
            />
          </li>
        </ul>
      </div>

      <WelcomeVaults
        :vaults="vaults"
        :heading="heading"
        :offer="offer"
        @opens="$emit('opens', $event)"
        @offers="$emit('offers')"
      >
        <template v-if="$slots.waiting" #waiting>
          <slot name="waiting" />
        </template>
        <template v-if="$slots.vault" #vault="{ vault }">
          <slot name="vault" :vault="vault" />
        </template>
      </WelcomeVaults>
    </div>

    <span class="welcome-page__version">{{ version }}</span>
  </div>
</template>

<style scoped>
/* One column in the middle of the window, held to the width of a short line so
   the rows read as a list and not as a page. Short of the height that column
   needs, it stands as two. */
.welcome-page {
  /* How tall the glyph stands over the name. */
  --glyph: 5.4rem;

  position: relative;
  display: flex;
  block-size: 100%;
  overflow: auto;
  container-type: size;
  padding: var(--numen-gutter);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
}

/* What build this is, in the far corner, where the column never reaches. */
.welcome-page__version {
  position: absolute;
  inset-block-end: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}

/* The column sits in the middle of whatever room there is. It is centred by its
   own margins, so a column taller than the room keeps its head inside it and
   the whole of it is scrolled to. */
.welcome-page__column {
  display: flex;
  flex-direction: column;
  gap: 1.4rem;
  margin: auto;
  inline-size: 100%;
  max-inline-size: 22rem;
}

/* The mark, the name and the ways in stand together, and take a column of their
   own where the screen has two. */
.welcome-page__lead {
  display: flex;
  flex-direction: column;
  gap: 1.4rem;
}

.welcome-page__head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

/* The glyph carries no size of its own, so it stands at the height it is given
   and is never squeezed to fit the room. */
.welcome-page__glyph {
  flex: none;
  inline-size: auto;
  block-size: var(--glyph);
}

/* The name is set in the letters the mark is drawn in, which are a serif's.
   Nothing else in the window is, so the family is this screen's own. */
.welcome-page__name {
  margin: 0;
  /* The glyph above is an italic and leans right, so the name sits a little
     left of the middle to read as under it. */
  margin-inline-end: 0.22em;
  font-family: ui-serif, Georgia, 'Times New Roman', serif;
  font-size: var(--numen-display-size);
  font-weight: 400;
  letter-spacing: 0.06em;
}

.welcome-page__ways {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

/* Short of the height the one column takes — the mark over the name, six ways
   in, the heading and two vaults — the ways in and the vaults stand side by
   side, each column the width of a short line. The list is the one thing here
   with no end to it, and the one thing that scrolls. */
@container (max-height: 27.55rem) and (min-width: 45.4rem) {
  .welcome-page__column {
    flex-direction: row;
    margin-block: 0;
    block-size: 100%;
    max-inline-size: 45.4rem;
  }

  /* The ways in stand at their own height, whole, from the top of the screen
     down. */
  .welcome-page__lead {
    flex: 1;
    align-self: start;
    min-inline-size: 0;
  }
}

/* Shorter than the mark, the name and the six ways in take together, the screen
   gives up the emblem a step at a time: the mark at half its height, then the
   mark, then the name after it. A way in is never given up. */
@container (max-height: 19.85rem) {
  .welcome-page__head {
    --glyph: 2.7rem;
  }
}

@container (max-height: 17.15rem) {
  .welcome-page__glyph {
    display: none;
  }
}

@container (max-height: 13.95rem) {
  .welcome-page__head {
    display: none;
  }
}
</style>
