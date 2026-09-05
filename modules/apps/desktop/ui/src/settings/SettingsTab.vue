<script setup lang="ts">
/**
 * The settings tab: everything in numen.json a person can change, grouped by
 * the part of the application it governs.
 *
 * A setting the window has a command for is drawn as a control and goes through
 * that command's own code; the rest are read out of the file and written back
 * where they stand.
 */
import { computed } from 'vue'
import { Button, NumberField, Segmented, Select, Switch, TimeField } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import type { Held } from './kind'
/**
 * The settings read out of the file whole, each as a path through it. A reading
 * off a scanned page and a transcript of a recording are put right at a profile
 * each, and each stands beside the thing it puts right.
 *
 * It stands in a file of its own because the core is held to it: a row reading
 * a key the settings file has no place for draws nothing, whatever is written.
 */
import AT from './paths.json'
import type { Mode } from '../theme'
import { INTERFACE_SCALE, MODE, TEXT_SCALE } from '../wearing'
import { choicesFor } from './models'
import { write } from './json5'
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
const themes = computed<readonly SelectChoice[]>(() =>
  [...held.value.themes()]
    .sort((one, other) => Number(other.shipped) - Number(one.shipped))
    .map((one) => ({
      id: one.name,
      text: one.title,
      group: one.shipped ? words.shipped : words.owned,
    })),
)

/** How fine a size may be turned, which is where the ladder of them steps. */
const STEP = 0.1

/** How early in the day a day of review may be asked to begin. */
const EARLIEST = '00:00'


/**
 * The ends the two fields type between. A field settles on the number inside
 * its ends, so an end the settings do not hold is a number of the person's cut
 * down behind them: the floors are the core's own, and above them the settings
 * take any whole number there is.
 *
 * Below nothing megabytes is no limit at all, and no steps at all is the
 * default number of them, which is what each setting reads its floor as.
 */
const WHOLE = Number.MAX_SAFE_INTEGER
const UNDER = { least: -1, most: WHOLE }
const STEPS = { least: 1, most: WHOLE }

/** What stands at a setting, read as the kind the row draws it as. */
const said = (at: readonly string[]): string => {
  const value = held.value.setting(at)
  return typeof value === 'string' ? value : ''
}
const on = (at: readonly string[]): boolean => held.value.setting(at) === true
const counted = (at: readonly string[]): number | null => {
  const value = held.value.setting(at)
  return typeof value === 'number' ? value : null
}

/** The models a setting can be set to: what the file holds, then the presets. */
const models = (at: readonly string[]): readonly SelectChoice[] =>
  choicesFor(held.value.models(at), said(at), words)

/**
 * A model chosen. A preset writes everything that preset decides; a value the
 * presets do not name is written where it stands.
 */
const picks = (at: readonly string[], name: string): void => {
  const model = held.value.models(at).find((one) => one.name === name)
  if (model) held.value.writes(model.writes)
  else puts(at, name)
}

/** One setting written, by what is to stand there. */
const puts = (at: readonly string[], value: unknown): void =>
  held.value.writes([{ at, value: write(value) }])

/** The profiles a reading may be put right at, and naming none. */
const profiles = computed<readonly SelectChoice[]>(() => {
  const kept = held.value.setting(AT.profiles)
  const names = kept && typeof kept === 'object' ? Object.keys(kept) : []
  return [{ id: '', text: words.proofreadingNone }, ...names.map((one) => ({ id: one, text: one }))]
})
</script>

<template>
  <div class="settings">
    <div class="settings__page">
      <!-- Where the settings stand, and the one way to the file itself. Every
           setting with a control is turned by its control. -->
      <div class="settings__where">
        <p class="settings__file">{{ held.file() || words.file }}</p>
        <Button variant="outline" size="small" @click="held.opensFile()">{{ words.opens }}</Button>
      </div>

      <section class="settings__group" :aria-label="words.window">
        <h2 class="settings__heading">{{ words.window }}</h2>

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-theme">{{ words.theme }}</label>
            <span class="settings__detail">{{ words.themeDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-theme"
              :model-value="held.applied()"
              :choices="themes"
              :name="words.theme"
              class="settings__choice"
              @update:model-value="(name: string) => held.chooses(name)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-mode">{{ words.mode }}</span>
            <span class="settings__detail">
              {{ held.pinned() ? words.pinned : words.modeDetail }}
            </span>
          </span>
          <span class="settings__value">
            <Segmented
              :model-value="held.mode()"
              :choices="modes"
              :disabled="held.pinned()"
              :aria-labelledby="'settings-mode'"
              @update:model-value="(one: string) => held.chooses(`${MODE}:${one as Mode}`)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-interface">{{ words.interfaceScale }}</span>
            <span class="settings__detail">{{ words.interfaceScaleDetail }}</span>
          </span>
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
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-text">{{ words.textScale }}</span>
            <span class="settings__detail">{{ words.textScaleDetail }}</span>
          </span>
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
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-hanging">{{ words.hanging }}</span>
            <span class="settings__detail">{{ words.hangingDetail }}</span>
          </span>
          <span class="settings__value">
            <Switch
              :model-value="held.hangs()"
              :aria-labelledby="'settings-hanging'"
              @update:model-value="(on: boolean) => held.choosesHanging(on)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-parts">{{ words.parts }}</span>
            <span class="settings__detail">{{ words.partsDetail }}</span>
          </span>
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
        </div>
      </section>

      <section class="settings__group" :aria-label="words.naming">
        <h2 class="settings__heading">{{ words.naming }}</h2>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-syncing">{{ words.syncing }}</span>
            <span class="settings__detail">{{ words.syncingDetail }}</span>
          </span>
          <span class="settings__value">
            <Switch
              :model-value="held.syncing()"
              :aria-labelledby="'settings-syncing'"
              @update:model-value="(on: boolean) => held.choosesSyncing(on)"
            />
          </span>
        </div>
      </section>

      <section class="settings__group" :aria-label="words.review">
        <h2 class="settings__heading">{{ words.review }}</h2>

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-day-starts">{{ words.dayStarts }}</label>
            <span class="settings__detail">{{ words.dayStartsDetail }}</span>
          </span>
          <span class="settings__value">
            <TimeField
              id="settings-day-starts"
              :model-value="held.dayStarts()"
              :min="EARLIEST"
              :max="held.latestDayStarts()"
              class="settings__number"
              @settles="(hour: string) => held.choosesDayStarts(hour)"
            />
          </span>
        </div>
      </section>

      <section class="settings__group" :aria-label="words.transcription">
        <h2 class="settings__heading">{{ words.transcription }}</h2>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-transcribing">{{ words.transcribing }}</span>
            <span class="settings__detail">{{ words.transcribingDetail }}</span>
          </span>
          <span class="settings__value">
            <Switch
              :model-value="on(AT.transcribing)"
              :aria-labelledby="'settings-transcribing'"
              @update:model-value="(kept: boolean) => puts(AT.transcribing, kept)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-under">{{ words.transcribeUnder }}</span>
            <span class="settings__detail">{{ words.transcribeUnderDetail }}</span>
          </span>
          <span class="settings__value">
            <NumberField
              :model-value="counted(AT.transcribeUnder)"
              :min="UNDER.least"
              :max="UNDER.most"
              :step="1"
              class="settings__number"
              :aria-labelledby="'settings-under'"
              @settles="(size: number | null) => size !== null && puts(AT.transcribeUnder, size)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-transcript-proofread">
              {{ words.transcriptProofread }}
            </label>
            <span class="settings__detail">{{ words.transcriptProofreadDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-transcript-proofread"
              :model-value="said(AT.transcriptProofread)"
              :choices="profiles"
              :name="words.transcriptProofread"
              class="settings__choice"
              @update:model-value="(name: string) => puts(AT.transcriptProofread, name)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-transcript-always">
              {{ words.transcriptProofreadAlways }}
            </span>
            <span class="settings__detail">{{ words.transcriptProofreadAlwaysDetail }}</span>
          </span>
          <span class="settings__value">
            <Switch
              :model-value="on(AT.transcriptProofreadAlways)"
              :aria-labelledby="'settings-transcript-always'"
              @update:model-value="(kept: boolean) => puts(AT.transcriptProofreadAlways, kept)"
            />
          </span>
        </div>
      </section>

      <section class="settings__group" :aria-label="words.ocr">
        <h2 class="settings__heading">{{ words.ocr }}</h2>

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-ocr">{{ words.ocrModel }}</label>
            <span class="settings__detail">{{ words.ocrModelDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-ocr"
              :model-value="said(AT.ocrModel)"
              :choices="models(AT.ocrModel)"
              :name="words.ocrModel"
              class="settings__choice"
              @update:model-value="(name: string) => picks(AT.ocrModel, name)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-ocr-proofread">
              {{ words.ocrProofread }}
            </label>
            <span class="settings__detail">{{ words.ocrProofreadDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-ocr-proofread"
              :model-value="said(AT.ocrProofread)"
              :choices="profiles"
              :name="words.ocrProofread"
              class="settings__choice"
              @update:model-value="(name: string) => puts(AT.ocrProofread, name)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-ocr-always">
              {{ words.ocrProofreadAlways }}
            </span>
            <span class="settings__detail">{{ words.ocrProofreadAlwaysDetail }}</span>
          </span>
          <span class="settings__value">
            <Switch
              :model-value="on(AT.ocrProofreadAlways)"
              :aria-labelledby="'settings-ocr-always'"
              @update:model-value="(kept: boolean) => puts(AT.ocrProofreadAlways, kept)"
            />
          </span>
        </div>
      </section>

      <section class="settings__group" :aria-label="words.indexing">
        <h2 class="settings__heading">{{ words.indexing }}</h2>

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-indexing">{{ words.indexingModel }}</label>
            <span class="settings__detail">{{ words.indexingModelDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-indexing"
              :model-value="said(AT.indexingModel)"
              :choices="models(AT.indexingModel)"
              :name="words.indexingModel"
              class="settings__choice"
              @update:model-value="(name: string) => picks(AT.indexingModel, name)"
            />
          </span>
        </div>
      </section>

      <section class="settings__group" :aria-label="words.agent">
        <h2 class="settings__heading">{{ words.agent }}</h2>

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-agent">{{ words.agentUse }}</label>
            <span class="settings__detail">{{ words.agentUseDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-agent"
              :model-value="said(AT.agent)"
              :choices="models(AT.agent)"
              :name="words.agentUse"
              class="settings__choice"
              @update:model-value="(name: string) => picks(AT.agent, name)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-agent-model">{{ words.agentModel }}</label>
            <span class="settings__detail">{{ words.agentModelDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-agent-model"
              :model-value="said(AT.agentModel)"
              :choices="models(AT.agentModel)"
              :name="words.agentModel"
              class="settings__choice"
              @update:model-value="(name: string) => picks(AT.agentModel, name)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-agent-steps">{{ words.agentSteps }}</span>
            <span class="settings__detail">{{ words.agentStepsDetail }}</span>
          </span>
          <span class="settings__value">
            <NumberField
              :model-value="counted(AT.agentSteps)"
              :min="STEPS.least"
              :max="STEPS.most"
              :step="1"
              class="settings__number"
              :aria-labelledby="'settings-agent-steps'"
              @settles="(count: number | null) => count !== null && puts(AT.agentSteps, count)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-agent-tools">{{ words.agentTools }}</span>
            <span class="settings__detail">{{ words.agentToolsDetail }}</span>
          </span>
          <span class="settings__value">
            <Switch
              :model-value="on(AT.agentTools)"
              :aria-labelledby="'settings-agent-tools'"
              @update:model-value="(kept: boolean) => puts(AT.agentTools, kept)"
            />
          </span>
        </div>

        <div class="settings__row">
          <span class="settings__said">
            <span class="settings__name" id="settings-agent-hooks">{{ words.agentHooks }}</span>
            <span class="settings__detail">{{ words.agentHooksDetail }}</span>
          </span>
          <span class="settings__value">
            <Switch
              :model-value="on(AT.agentHooks)"
              :aria-labelledby="'settings-agent-hooks'"
              @update:model-value="(kept: boolean) => puts(AT.agentHooks, kept)"
            />
          </span>
        </div>

      </section>
    </div>
  </div>
</template>

<style scoped>
.settings {
  /* The measure the settings are read at. It is wide enough for a model to be
     read at the end of a row it is chosen on. */
  --settings-measure: 54rem;
  /* Between one group and the next, and between a heading and its rows. */
  --settings-apart: 1.75rem;
  --settings-near: 0.375rem;
  /* One row: the box a number is typed into, the box a choice is taken from,
     the air around the row, and the space between what it is called and what
     it means. */
  --settings-value: 6rem;
  --settings-choice: 18rem;
  --settings-row-air: 0.5rem;
  --settings-said-gap: 0.125rem;
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

/* Where the settings stand, with the one way to the file itself at the end of
   the line the groups are read against. */
.settings__where {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap);
  max-inline-size: var(--settings-measure);
  margin: 0 auto var(--settings-apart);
}

.settings__file {
  min-inline-size: 0;
  margin: 0;
  color: var(--numen-hushed);
  overflow-wrap: anywhere;
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

/* The name and what it means on the left, the control at the end of the row. */
.settings__row {
  display: grid;
  grid-template-columns: 1fr max-content;
  align-items: center;
  gap: 0 var(--numen-panel-gap);
  padding-block: var(--settings-row-air);
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
}

/* What the row is called, and under it what it means. */
.settings__said {
  display: flex;
  flex-direction: column;
  gap: var(--settings-said-gap);
  min-inline-size: 0;
}

/* Controls of every width end at the one edge. */
.settings__value {
  display: flex;
  align-items: center;
  justify-content: end;
}

.settings__number {
  inline-size: var(--settings-value);
}

.settings__choice {
  inline-size: var(--settings-choice);
}

.settings__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
