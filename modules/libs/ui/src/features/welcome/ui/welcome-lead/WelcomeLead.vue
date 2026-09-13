<script setup lang="ts">
/**
 * The mark, the name under it, and the ways in below them.
 *
 * A window offering no way in draws none, and the mark gives way a step at a
 * time as the screen shortens.
 */
import { Glyph } from '../glyph'
import { WelcomeRow } from '../welcome-row'
import type { WelcomeAction } from '../../lib/welcome'

withDefaults(
  defineProps<{
    /** What stands under the mark, telling a person which window they opened. */
    name?: string
    /** The ways in, above the list. */
    ways?: readonly WelcomeAction[]
  }>(),
  { name: 'numen', ways: () => [] },
)

defineEmits<{
  /** A way chosen, by the identifier the caller gave it. */
  (event: 'run', id: string): void
}>()
</script>

<template>
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
          @click="$emit('run', one.id)"
        />
      </li>
    </ul>
  </div>
</template>

<style scoped>
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

/* Where the screen stands as two columns, the ways in stand at their own
   height, whole, from the top of the screen down. */
@container (max-height: 27.55rem) and (min-width: 45.4rem) {
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
