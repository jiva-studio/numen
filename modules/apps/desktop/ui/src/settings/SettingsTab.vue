<script setup lang="ts">
/**
 * The settings tab: everything in numen.json a person can change, in the
 * groups the file keeps them in.
 *
 * A setting the window has a command for is drawn as a control and goes through
 * the same code that command goes through. The rest are read out of the file
 * and written back into it where they stand. A setting that is an object or a
 * list is opened in the editor, written as JSON5.
 */
import { computed } from 'vue'
import { NumberField, Segmented, Select, Switch, TimeField, type SelectChoice } from '@numen/ui'
import Editable from './Editable.vue'
import type { Held } from './kind'
import type { Mode } from '../theme'
import { LATEST_STARTS as LATEST } from '../reviewing'
import { INTERFACE_SCALE, MODE, TEXT_SCALE } from '../wearing'
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

/** The settings read out of the file whole, each as a path through it. */
const AT = {
  indexingModel: ['indexing', 'embedding', 'model', 'name'],
  indexing: ['indexing', 'embedding'],
  ocrModel: ['indexing', 'recognition', 'recognise', 'name'],
  ocr: ['indexing', 'recognition'],
  proofreadWith: ['indexing', 'recognition', 'proofread', 'with'],
  profiles: ['indexing', 'proofreading', 'profiles'],
  proofreading: ['indexing', 'proofreading'],
  transcribing: ['indexing', 'transcribe_recordings'],
  transcribeUnder: ['indexing', 'transcribe_under_mb'],
  transcription: ['indexing', 'transcription'],
  agent: ['agent', 'use'],
  agentModel: ['agent', 'claude', 'model'],
  agentSteps: ['agent', 'claude', 'max_steps'],
  agentTools: ['agent', 'serve_tools'],
  agentHooks: ['agent', 'claude', 'reads_hooks_and_skills'],
  agentCommand: ['agent', 'claude', 'command'],
} as const

/**
 * The ends the two fields type between. The settings take any whole number, and
 * these are as far as a hand is asked to turn one. Below nothing megabytes are
 * no limit at all, which is what the setting reads a negative number as.
 */
const UNDER = { least: -1, most: 100000 }
const STEPS = { least: 1, most: 200 }

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

/**
 * The models a setting can be set to. A name the file holds that this build
 * does not offer is offered all the same, and is drawn as one that is not there.
 */
const models = (at: readonly string[]): readonly SelectChoice[] => {
  const offered = held.value.models(at).map((one) => ({
    id: one.name,
    text: one.byDefault ? `${one.title} — ${words.byDefault}` : one.title,
    ...(one.shelf ? { group: one.shelf } : {}),
  }))
  const now = said(at)
  if (offered.some((one) => one.id === now)) return offered
  return [...offered, { id: now, text: `${now} — ${words.notFound}` }]
}

/** A model chosen, which writes everything that model decides. */
const picks = (at: readonly string[], name: string): void => {
  const model = held.value.models(at).find((one) => one.name === name)
  if (model) held.value.writes(model.writes)
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
      <p class="settings__where">{{ held.file() || words.file }}</p>

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
              :max="LATEST"
              class="settings__number"
              @settles="(hour: string) => held.choosesDayStarts(hour)"
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
              class="settings__choice"
              @update:model-value="(name: string) => picks(AT.indexingModel, name)"
            />
          </span>
        </div>

        <Editable
          :name="words.indexingSection"
          :detail="words.indexingSectionDetail"
          :value="held.setting(AT.indexing)"
          @keeps="(value: unknown) => puts(AT.indexing, value)"
        />

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-ocr">{{ words.ocr }}</label>
            <span class="settings__detail">{{ words.ocrDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-ocr"
              :model-value="said(AT.ocrModel)"
              :choices="models(AT.ocrModel)"
              class="settings__choice"
              @update:model-value="(name: string) => picks(AT.ocrModel, name)"
            />
          </span>
        </div>

        <Editable
          :name="words.ocrSection"
          :detail="words.ocrSectionDetail"
          :value="held.setting(AT.ocr)"
          @keeps="(value: unknown) => puts(AT.ocr, value)"
        />

        <div class="settings__row">
          <span class="settings__said">
            <label class="settings__name" for="settings-proofreading">
              {{ words.proofreading }}
            </label>
            <span class="settings__detail">{{ words.proofreadingDetail }}</span>
          </span>
          <span class="settings__value">
            <Select
              id="settings-proofreading"
              :model-value="said(AT.proofreadWith)"
              :choices="profiles"
              class="settings__choice"
              @update:model-value="(name: string) => puts(AT.proofreadWith, name)"
            />
          </span>
        </div>

        <Editable
          :name="words.proofreadingSection"
          :detail="words.proofreadingSectionDetail"
          :value="held.setting(AT.proofreading)"
          @keeps="(value: unknown) => puts(AT.proofreading, value)"
        />

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

        <Editable
          :name="words.transcriptionSection"
          :detail="words.transcriptionSectionDetail"
          :value="held.setting(AT.transcription)"
          @keeps="(value: unknown) => puts(AT.transcription, value)"
        />
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

        <Editable
          :name="words.agentCommand"
          :detail="words.agentCommandDetail"
          :value="held.setting(AT.agentCommand)"
          @keeps="(value: unknown) => puts(AT.agentCommand, value)"
        />
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
