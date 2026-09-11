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
import { Button, NumberField, SegmentedControl, Select, Switch, TimeField } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import type { SettingsTabState } from './kind'
/**
 * The settings read out of the file whole, each as a path through it. A reading
 * off a scanned page and a transcript of a recording are put right at a profile
 * each, and each stands beside the thing it puts right.
 *
 * It stands in a file of its own because the core is held to it: a row reading
 * a key the settings file has no place for draws nothing, whatever is written.
 */
import AT from './paths.json'
import type { Mode } from '../shared/settings/theme'
import { INTERFACE_SCALE, MODE, TEXT_SCALE } from '../shared/settings/appearance'
import { choicesFor } from './models'
import { write } from '../shared/settings/write'
import { WORDS as words } from './words'

// --- Props & Emits ---
const props = defineProps<{ state: SettingsTabState }>()

// --- State ---
const installation = computed(() => props.state.installation)

/** How large the interface may be drawn, and how large the text may be set. */
const bounds = computed(() => installation.value.bounds.value)

/** The three halves of a colour pair, as the switch offers them. */
const modes = [
  { id: 'system', text: words.system },
  { id: 'light', text: words.light },
  { id: 'dark', text: words.dark },
]

/** The themes, in the two shelves they come off. */
const themes = computed<readonly SelectChoice[]>(() =>
  [...installation.value.themes.value]
    .sort(
      (one, other) =>
        Number(other.isBuiltIn ?? other.shipped) - Number(one.isBuiltIn ?? one.shipped),
    )
    .map((one) => ({
      id: one.name,
      text: one.title,
      group: (one.isBuiltIn ?? one.shipped) ? words.shipped : words.owned,
    })),
)

/** How fine a size may be turned, which is where the ladder of them steps. */
const STEP = 0.1

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

/** The profiles a reading may be put right at, and naming none. */
const profiles = computed<readonly SelectChoice[]>(() => {
  const kept = installation.value.setting(AT.profiles)
  const names = kept && typeof kept === 'object' ? Object.keys(kept) : []
  return [{ id: '', text: words.proofreadingNone }, ...names.map((one) => ({ id: one, text: one }))]
})

// --- Handlers ---
function onOpenFile() {
  installation.value.opensFile()
}

function onThemeChange(name: string) {
  installation.value.chooses(name)
}

function onModeChange(modeChoice: string) {
  installation.value.chooses(`${MODE}:${modeChoice as Mode}`)
}

function onInterfaceScaleChange(size: number | null) {
  if (size !== null) {
    installation.value.chooses(`${INTERFACE_SCALE}:${size}`)
  }
}

function onTextScaleChange(size: number | null) {
  if (size !== null) {
    installation.value.chooses(`${TEXT_SCALE}:${size}`)
  }
}

function onPartsChange(count: number | null) {
  if (count !== null) {
    installation.value.choosesParts(count)
  }
}

function onDayStartsChange(hour: string) {
  installation.value.choosesDayStarts(hour)
}

function onModelChange(at: readonly string[], name: string) {
  const model = installation.value.models(at).find((one) => one.name === name)
  if (model) installation.value.writes(model.writes)
  else setSetting(at, name)
}

function onSettingChange(at: readonly string[], value: unknown) {
  setSetting(at, value)
}

// --- Helpers ---
function getSettingString(at: readonly string[]): string {
  const value = installation.value.setting(at)
  return typeof value === 'string' ? value : ''
}
const said = getSettingString

function isSettingEnabled(at: readonly string[]): boolean {
  return installation.value.setting(at) === true
}
const on = isSettingEnabled

function getSettingNumber(at: readonly string[]): number | null {
  const value = installation.value.setting(at)
  return typeof value === 'number' ? value : null
}
const counted = getSettingNumber

function getModels(at: readonly string[]): readonly SelectChoice[] {
  return choicesFor(installation.value.models(at), getSettingString(at), words)
}
const models = getModels

function setSetting(at: readonly string[], value: unknown): void {
  installation.value.writes([{ at, value: write(value) }])
}
</script>

<template>
  <div class="settings">
    <div class="settings__page">
      <!-- Where the settings stand, and the one way to the file itself. Every
           setting with a control is turned by its control. -->
      <div class="settings__where">
        <p class="settings__file">{{ installation.file.value || words.file }}</p>
        <Button variant="outline" size="small" @click="onOpenFile">
          {{ words.opens }}
        </Button>
      </div>

      <section class="settings__group" :aria-label="words.window">
        <h2 class="settings__heading">{{ words.window }}</h2>

        <SettingRow
          v-slot="{ labelledBy }"
          at="theme"
          :name="words.theme"
          :detail="words.themeDetail"
        >
          <Select
            :model-value="installation.applied.value"
            :choices="themes"
            :name="words.theme"
            :aria-labelledby="labelledBy"
            class="settings__choice"
            @update:model-value="onThemeChange"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="mode"
          :name="words.mode"
          :detail="installation.pinned.value ? words.pinned : words.modeDetail"
        >
          <SegmentedControl
            :model-value="installation.mode.value"
            :choices="modes"
            :disabled="installation.pinned.value"
            :aria-labelledby="labelledBy"
            @update:model-value="onModeChange"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="interface"
          :name="words.interfaceScale"
          :detail="words.interfaceScaleDetail"
        >
          <NumberField
            :model-value="installation.sizes.value.interfaceScale"
            :min="bounds.interfaceScale.least"
            :max="bounds.interfaceScale.most"
            :step="STEP"
            :aria-labelledby="labelledBy"
            class="settings__number"
            @update:model-value="onInterfaceScaleChange"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="text"
          :name="words.textScale"
          :detail="words.textScaleDetail"
        >
          <NumberField
            :model-value="installation.sizes.value.textScale"
            :min="bounds.textScale.least"
            :max="bounds.textScale.most"
            :step="STEP"
            :aria-labelledby="labelledBy"
            class="settings__number"
            @update:model-value="onTextScaleChange"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="hanging"
          :name="words.hanging"
          :detail="words.hangingDetail"
        >
          <Switch v-model="installation.hangs.value" :aria-labelledby="labelledBy" />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="parts"
          :name="words.parts"
          :detail="words.partsDetail"
        >
          <NumberField
            :model-value="installation.parts.value"
            :min="installation.partsBounds.value.least"
            :max="installation.partsBounds.value.most"
            :step="1"
            :aria-labelledby="labelledBy"
            class="settings__number"
            @update:model-value="onPartsChange"
          />
        </SettingRow>
      </section>

      <section class="settings__group" :aria-label="words.naming">
        <h2 class="settings__heading">{{ words.naming }}</h2>

        <SettingRow
          v-slot="{ labelledBy }"
          at="syncing"
          :name="words.syncing"
          :detail="words.syncingDetail"
        >
          <Switch v-model="installation.syncing.value" :aria-labelledby="labelledBy" />
        </SettingRow>
      </section>

      <section class="settings__group" :aria-label="words.review">
        <h2 class="settings__heading">{{ words.review }}</h2>

        <SettingRow
          v-slot="{ labelledBy }"
          at="day-starts"
          :name="words.dayStarts"
          :detail="words.dayStartsDetail"
        >
          <TimeField
            :model-value="installation.dayStarts.value"
            :max="installation.latestDayStarts.value"
            :aria-labelledby="labelledBy"
            class="settings__number"
            @settles="onDayStartsChange"
          />
        </SettingRow>
      </section>

      <section class="settings__group" :aria-label="words.transcription">
        <h2 class="settings__heading">{{ words.transcription }}</h2>

        <SettingRow
          v-slot="{ labelledBy }"
          at="transcribing"
          :name="words.transcribing"
          :detail="words.transcribingDetail"
        >
          <Switch
            :model-value="on(AT.transcribing)"
            :aria-labelledby="labelledBy"
            @update:model-value="(kept: boolean) => onSettingChange(AT.transcribing, kept)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="under"
          :name="words.transcribeUnder"
          :detail="words.transcribeUnderDetail"
        >
          <NumberField
            :model-value="counted(AT.transcribeUnder)"
            :min="UNDER.least"
            :max="UNDER.most"
            :step="1"
            :aria-labelledby="labelledBy"
            class="settings__number"
            @settles="(size: number | null) => size !== null && onSettingChange(AT.transcribeUnder, size)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="transcript-proofread"
          :name="words.transcriptProofread"
          :detail="words.transcriptProofreadDetail"
        >
          <Select
            :model-value="said(AT.transcriptProofread)"
            :choices="profiles"
            :name="words.transcriptProofread"
            :aria-labelledby="labelledBy"
            class="settings__choice"
            @update:model-value="(name: string) => onSettingChange(AT.transcriptProofread, name)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="transcript-always"
          :name="words.transcriptProofreadAlways"
          :detail="words.transcriptProofreadAlwaysDetail"
        >
          <Switch
            :model-value="on(AT.transcriptProofreadAlways)"
            :aria-labelledby="labelledBy"
            @update:model-value="(kept: boolean) => onSettingChange(AT.transcriptProofreadAlways, kept)"
          />
        </SettingRow>
      </section>

      <section class="settings__group" :aria-label="words.ocr">
        <h2 class="settings__heading">{{ words.ocr }}</h2>

        <SettingRow
          v-slot="{ labelledBy }"
          at="ocr"
          :name="words.ocrModel"
          :detail="words.ocrModelDetail"
        >
          <Select
            :model-value="said(AT.ocrModel)"
            :choices="models(AT.ocrModel)"
            :name="words.ocrModel"
            :aria-labelledby="labelledBy"
            class="settings__choice"
            @update:model-value="(name: string) => onModelChange(AT.ocrModel, name)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="ocr-proofread"
          :name="words.ocrProofread"
          :detail="words.ocrProofreadDetail"
        >
          <Select
            :model-value="said(AT.ocrProofread)"
            :choices="profiles"
            :name="words.ocrProofread"
            :aria-labelledby="labelledBy"
            class="settings__choice"
            @update:model-value="(name: string) => onSettingChange(AT.ocrProofread, name)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="ocr-always"
          :name="words.ocrProofreadAlways"
          :detail="words.ocrProofreadAlwaysDetail"
        >
          <Switch
            :model-value="on(AT.ocrProofreadAlways)"
            :aria-labelledby="labelledBy"
            @update:model-value="(kept: boolean) => onSettingChange(AT.ocrProofreadAlways, kept)"
          />
        </SettingRow>
      </section>

      <section class="settings__group" :aria-label="words.indexing">
        <h2 class="settings__heading">{{ words.indexing }}</h2>

        <SettingRow
          v-slot="{ labelledBy }"
          at="indexing"
          :name="words.indexingModel"
          :detail="words.indexingModelDetail"
        >
          <Select
            :model-value="said(AT.indexingModel)"
            :choices="models(AT.indexingModel)"
            :name="words.indexingModel"
            :aria-labelledby="labelledBy"
            class="settings__choice"
            @update:model-value="(name: string) => onModelChange(AT.indexingModel, name)"
          />
        </SettingRow>
      </section>

      <section class="settings__group" :aria-label="words.agent">
        <h2 class="settings__heading">{{ words.agent }}</h2>

        <SettingRow
          v-slot="{ labelledBy }"
          at="agent"
          :name="words.agentUse"
          :detail="words.agentUseDetail"
        >
          <Select
            :model-value="said(AT.agent)"
            :choices="models(AT.agent)"
            :name="words.agentUse"
            :aria-labelledby="labelledBy"
            class="settings__choice"
            @update:model-value="(name: string) => onModelChange(AT.agent, name)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="agent-model"
          :name="words.agentModel"
          :detail="words.agentModelDetail"
        >
          <Select
            :model-value="said(AT.agentModel)"
            :choices="models(AT.agentModel)"
            :name="words.agentModel"
            :aria-labelledby="labelledBy"
            class="settings__choice"
            @update:model-value="(name: string) => onModelChange(AT.agentModel, name)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="agent-steps"
          :name="words.agentSteps"
          :detail="words.agentStepsDetail"
        >
          <NumberField
            :model-value="counted(AT.agentSteps)"
            :min="STEPS.least"
            :max="STEPS.most"
            :step="1"
            :aria-labelledby="labelledBy"
            class="settings__number"
            @settles="(count: number | null) => count !== null && onSettingChange(AT.agentSteps, count)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="agent-tools"
          :name="words.agentTools"
          :detail="words.agentToolsDetail"
        >
          <Switch
            :model-value="on(AT.agentTools)"
            :aria-labelledby="labelledBy"
            @update:model-value="(kept: boolean) => onSettingChange(AT.agentTools, kept)"
          />
        </SettingRow>

        <SettingRow
          v-slot="{ labelledBy }"
          at="agent-hooks"
          :name="words.agentHooks"
          :detail="words.agentHooksDetail"
        >
          <Switch
            :model-value="on(AT.agentHooks)"
            :aria-labelledby="labelledBy"
            @update:model-value="(kept: boolean) => onSettingChange(AT.agentHooks, kept)"
          />
        </SettingRow>
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
  /* On a row: the box a number is typed into, and the box a choice is taken
     from. */
  --settings-value: 6rem;
  --settings-choice: 18rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
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

.settings__number {
  inline-size: var(--settings-value);
}

.settings__choice {
  inline-size: var(--settings-choice);
}
</style>
