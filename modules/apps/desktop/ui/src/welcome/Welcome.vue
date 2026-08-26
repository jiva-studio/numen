<script setup lang="ts">
/**
 * What the window draws while it holds nothing open.
 *
 * The mark and the name, the ways into the vault under the keystrokes that
 * reach them, and the vaults this installation holds.
 */
import { KeyCap } from '@numen/ui'
import { FolderRoot } from '@lucide/vue'
import { iconFor } from '../icons'
import Mark from './Mark.vue'
import type { Held, Way, Words } from './welcoming'

defineProps<{ ways: readonly Way[]; vaults: readonly Held[]; words: Words }>()
defineEmits<{
  /** A way chosen. `COMMANDS` means put the commands up; anything else is a command asked for. */
  (event: 'runs', id: string): void
  /** A vault on the list chosen, which the window opens. */
  (event: 'opens', id: string): void
  /** Another vault asked for. */
  (event: 'adds'): void
}>()
</script>

<template>
  <div class="welcome">
    <div class="welcome__column">
      <div class="welcome__head">
        <Mark class="welcome__mark" />
        <h1 class="welcome__name">numen</h1>
      </div>

      <ul v-if="ways.length" class="welcome__ways">
        <li v-for="one in ways" :key="one.id">
          <button type="button" class="welcome__row" @click="$emit('runs', one.id)">
            <component :is="iconFor(one.id)" v-if="iconFor(one.id)" class="welcome__icon" />
            <span class="welcome__what">{{ one.text }}</span>
            <KeyCap v-if="one.keys" :keys="one.keys" />
          </button>
        </li>
      </ul>

      <section class="welcome__vaults">
        <h2 class="welcome__heading">{{ words.vaults }}</h2>
        <ul class="welcome__list">
          <li v-for="one in vaults" :key="one.id">
            <button
              type="button"
              class="welcome__row welcome__row--vault"
              @click="$emit('opens', one.id)"
            >
              <FolderRoot class="welcome__icon" />
              <span class="welcome__named">
                <span class="welcome__what">{{ one.name }}</span>
                <!-- The whole path is on the element, for one too long to be drawn. -->
                <span class="welcome__aside" :title="one.path">{{ one.path }}</span>
              </span>
              <span v-if="one.detail" class="welcome__state">{{ one.detail }}</span>
            </button>
          </li>
        </ul>
        <button type="button" class="welcome__row" @click="$emit('adds')">
          <component :is="iconFor('newVault')" class="welcome__icon" />
          <span class="welcome__named">
            <span class="welcome__what">{{ words.newVault }}</span>
            <span class="welcome__aside">{{ words.newVaultDetail }}</span>
          </span>
        </button>
      </section>
    </div>
  </div>
</template>

<style scoped>
/* One column in the middle of the window, held to the width of a short line so
   the rows read as a list and not as a page. */
.welcome {
  /* How tall the glyph stands over the name. */
  --mark: 5.4rem;

  display: flex;
  align-items: center;
  justify-content: center;
  block-size: 100%;
  overflow: auto;
  padding: var(--numen-gutter);
  color: var(--numen-node-fg);
  font-family: var(--numen-font-sans);
  font-size: calc(var(--numen-font-size) * 13.6 / 13);
}

.welcome__column {
  display: flex;
  flex-direction: column;
  gap: 1.4rem;
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
  font-family: ui-serif, Georgia, 'Times New Roman', serif;
  font-size: calc(var(--numen-font-size) * 21 / 13);
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

.welcome__row:hover {
  background: var(--numen-bubble-bg);
}

.welcome__row:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: 1px;
}

/* A name is as long as a person makes it, and a long one ends in an ellipsis. */
.welcome__what {
  flex: 1;
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* One line, then an ellipsis: a path is as long as the machine makes it. */
.welcome__aside {
  overflow: hidden;
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* What is true of one row of the list and not of the ones beside it, said at
   the far end of it. */
.welcome__state {
  flex: none;
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
}

.welcome__heading {
  margin: 0 0 0.25rem;
  padding-inline: 0.6rem;
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  font-weight: inherit;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}
</style>
