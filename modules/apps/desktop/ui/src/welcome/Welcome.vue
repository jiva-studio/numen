<script setup lang="ts">
/**
 * What the window draws while it holds nothing open.
 *
 * The mark and the name, the ways into the vault under the keystrokes that
 * reach them, and the vaults this installation holds.
 */
import { KeyCap } from '@numen/ui'
import { Vault } from '@lucide/vue'
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
              <Vault class="welcome__icon" />
              <span class="welcome__named">
                <span class="welcome__what">{{ one.name }}</span>
                <span v-if="one.detail" class="welcome__aside">{{ one.detail }}</span>
              </span>
            </button>
          </li>
        </ul>
        <button type="button" class="welcome__row" @click="$emit('adds')">
          <component :is="iconFor('newVault')" class="welcome__icon" />
          <span class="welcome__what">{{ words.newVault }}</span>
        </button>
      </section>
    </div>
  </div>
</template>

<style scoped>
/* One column in the middle of the window, held to the width of a short line so
   the rows read as a list and not as a page. */
.welcome {
  /* How large the mark stands over the name. */
  --mark: 6rem;

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
  gap: 0.4rem;
}

.welcome__mark {
  inline-size: var(--mark);
  block-size: var(--mark);
}

.welcome__name {
  margin: 0;
  font-size: calc(var(--numen-font-size) * 24 / 13);
  font-weight: 400;
  letter-spacing: 0.04em;
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

/* A vault stands over what is true of it. */
/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.welcome__icon {
  flex: none;
  inline-size: 1rem;
  block-size: 1rem;
  stroke-width: 1.75;
  opacity: 0.75;
}

/* A vault is its name over what is said about it, beside the one icon. */
.welcome__named {
  display: flex;
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

.welcome__aside {
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
