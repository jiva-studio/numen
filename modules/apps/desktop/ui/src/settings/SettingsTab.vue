<script setup lang="ts">
/**
 * The settings tab: everything in numen.json a person can change, in the
 * groups the file keeps them in.
 *
 * A setting the window can write is drawn as a control and goes through the
 * same code the palette command of that name goes through. A setting the
 * window does not reach yet is drawn where it belongs and says where it stands.
 */
import { computed } from 'vue'
import { NumberField, Segmented, Switch } from '@numen/ui'
import type { Held } from './kind'
import type { Mode } from '../theme'
import { INTERFACE_SCALE, MODE, TEXT_SCALE } from '../wearing'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

const held = computed(() => props.held.installation)

/** How large the interface may be drawn, and how large the text may be set. */
const bounds = computed(() => held.value.bounds())

/** The three halves of a colour pair, as the switch offers them. */
const modes = [
  { id: 'system', text: words.system },
  { id: 'light', text: words.light },
  { id: 'dark', text: words.dark },
]

/** The themes, in the two shelves they come off. */
const shipped = computed(() => held.value.themes().filter((one) => one.shipped))
const owned = computed(() => held.value.themes().filter((one) => !one.shipped))

/** How fine a size may be turned, which is where the ladder of them steps. */
const STEP = 0.1

/** The settings the window shows and does not write. Each says where it stands. */
const elsewhere = [
  { group: words.review, name: words.dayStarts, detail: words.dayStartsDetail },
  { group: words.indexing, name: words.embedding, detail: words.embeddingDetail },
  { group: words.indexing, name: words.recognition, detail: words.recognitionDetail },
  { group: words.indexing, name: words.proofreading, detail: words.proofreadingDetail },
  { group: words.agent, name: words.agentUse, detail: words.agentUseDetail },
]

/** The groups those stand in, in the order they are drawn. */
const groups = [words.review, words.indexing, words.agent]

const reading = (group: string) => elsewhere.filter((one) => one.group === group)
</script>

<template>
  <div class="settings">
    <div class="settings__page">
      <p class="settings__where">{{ words.file }}</p>

      <section class="settings__group" :aria-label="words.window">
        <h2 class="settings__heading">{{ words.window }}</h2>

        <div class="settings__row">
          <label class="settings__name" for="settings-theme">{{ words.theme }}</label>
          <span class="settings__value">
            <select
              id="settings-theme"
              class="settings__select"
              :value="held.applied()"
              @change="held.chooses(($event.target as HTMLSelectElement).value)"
            >
              <optgroup :label="words.shipped">
                <option v-for="one in shipped" :key="one.name" :value="one.name">
                  {{ one.title }}
                </option>
              </optgroup>
              <optgroup v-if="owned.length" :label="words.owned">
                <option v-for="one in owned" :key="one.name" :value="one.name">
                  {{ one.title }}
                </option>
              </optgroup>
            </select>
          </span>
          <span class="settings__detail">{{ words.themeDetail }}</span>
        </div>

        <div class="settings__row">
          <span class="settings__name" id="settings-mode">{{ words.mode }}</span>
          <span class="settings__value">
            <Segmented
              :model-value="held.mode()"
              :choices="modes"
              :disabled="held.pinned()"
              :aria-labelledby="'settings-mode'"
              @update:model-value="(one: string) => held.chooses(`${MODE}:${one as Mode}`)"
            />
          </span>
          <span class="settings__detail">
            {{ held.pinned() ? words.pinned : words.modeDetail }}
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__name" id="settings-interface">{{ words.interfaceScale }}</span>
          <span class="settings__value">
            <NumberField
              :model-value="held.sizes().interfaceScale"
              :min="bounds.interfaceScale.least"
              :max="bounds.interfaceScale.most"
              :step="STEP"
              class="settings__number"
              :aria-labelledby="'settings-interface'"
              @update:model-value="
                (size: number | null) =>
                  size !== null && held.chooses(`${INTERFACE_SCALE}:${size}`)
              "
            />
          </span>
          <span class="settings__detail">{{ words.interfaceScaleDetail }}</span>
        </div>

        <div class="settings__row">
          <span class="settings__name" id="settings-text">{{ words.textScale }}</span>
          <span class="settings__value">
            <NumberField
              :model-value="held.sizes().textScale"
              :min="bounds.textScale.least"
              :max="bounds.textScale.most"
              :step="STEP"
              class="settings__number"
              :aria-labelledby="'settings-text'"
              @update:model-value="
                (size: number | null) => size !== null && held.chooses(`${TEXT_SCALE}:${size}`)
              "
            />
          </span>
          <span class="settings__detail">{{ words.textScaleDetail }}</span>
        </div>

        <div class="settings__row">
          <span class="settings__name" id="settings-hanging">{{ words.hanging }}</span>
          <span class="settings__value">
            <Switch
              :model-value="held.hangs()"
              :aria-labelledby="'settings-hanging'"
              @update:model-value="(on: boolean) => held.choosesHanging(on)"
            />
          </span>
          <span class="settings__detail">{{ words.hangingDetail }}</span>
        </div>

        <div class="settings__row">
          <span class="settings__name" id="settings-parts">{{ words.parts }}</span>
          <span class="settings__value">
            <NumberField
              :model-value="held.parts()"
              :min="1"
              :max="12"
              :step="1"
              class="settings__number"
              :aria-labelledby="'settings-parts'"
              @update:model-value="
                (count: number | null) => count !== null && held.choosesParts(count)
              "
            />
          </span>
          <span class="settings__detail">{{ words.partsDetail }}</span>
        </div>
      </section>

      <section class="settings__group" :aria-label="words.naming">
        <h2 class="settings__heading">{{ words.naming }}</h2>

        <div class="settings__row">
          <span class="settings__name" id="settings-syncing">{{ words.syncing }}</span>
          <span class="settings__value">
            <Switch
              :model-value="held.syncing()"
              :aria-labelledby="'settings-syncing'"
              @update:model-value="(on: boolean) => held.choosesSyncing(on)"
            />
          </span>
          <span class="settings__detail">{{ words.syncingDetail }}</span>
        </div>
      </section>

      <section v-for="group in groups" :key="group" class="settings__group" :aria-label="group">
        <h2 class="settings__heading">{{ group }}</h2>

        <div v-for="one in reading(group)" :key="one.name" class="settings__row">
          <span class="settings__name">{{ one.name }}</span>
          <span class="settings__value">
            <span class="settings__elsewhere">{{ words.inTheFile }}</span>
          </span>
          <span class="settings__detail">{{ one.detail }}</span>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.settings {
  /* The measure the settings are read at. */
  --settings-measure: 46rem;
  /* Between one group and the next, and between a heading and its rows. */
  --settings-apart: 1.75rem;
  --settings-near: 0.375rem;
  /* One row: its two set columns, and the air around it. */
  --settings-name: 12rem;
  --settings-value: 6rem;
  --settings-row-air: 0.5rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

.settings__page {
  flex: 1;
  min-block-size: 0;
  overflow-y: auto;
  padding: var(--numen-gutter);
}

.settings__where {
  max-inline-size: var(--settings-measure);
  margin: 0 auto var(--settings-apart);
  color: var(--numen-hushed);
}

/* The groups are read in one column, centred in whatever room the pane has. */
.settings__group {
  max-inline-size: var(--settings-measure);
  margin-block-end: var(--settings-apart);
  margin-inline: auto;
}

/* A label over the rows it names, set as this window sets its labels. */
.settings__heading {
  margin: 0 0 var(--settings-near);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.settings__row {
  display: grid;
  grid-template-columns: var(--settings-name) minmax(var(--settings-value), max-content) 1fr;
  align-items: baseline;
  gap: 0 var(--numen-panel-gap);
  padding-block: var(--settings-row-air);
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
}

.settings__value {
  display: flex;
  align-items: center;
}

.settings__number {
  inline-size: var(--settings-value);
}

.settings__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.settings__select {
  min-block-size: var(--numen-field-min);
  padding: 0 var(--numen-field-padding);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-field);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
}

.settings__select:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: var(--numen-stroke);
}

/* A setting this window shows and does not write. */
.settings__elsewhere {
  padding-inline: var(--numen-node-gap);
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-radius: var(--numen-radius-pill);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
