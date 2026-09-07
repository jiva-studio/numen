<script setup lang="ts">
/**
 * A note pointing at an address, drawn the way a recording is: what is at the
 * address plays at the top, and under it stands the transcript, which is the
 * text this tab writes.
 *
 * A time in the gutter is a moment in what is playing, so choosing one plays
 * from there. What can be asked over the address stands in the menu on the
 * player, each item only where it applies.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { Editor, Menu, timing } from '@numen/ui'
import type { Position } from '@numen/ui'
import { Ellipsis } from '@lucide/vue'
import LinkEmbed from './LinkEmbed.vue'
import { DOWNLOAD, DROP, FETCH, WORDS as words } from './words'
import type { NoteTabState } from './kind'

const props = defineProps<{ state: NoteTabState }>()

/** The words fetched for the address, and nothing until they are read. */
const transcript = computed(() => props.state.transcript.value)

const embed = useTemplateRef<{ seeks: (ms: number) => void }>('embed')

/** The times in the gutter, each playing what is at the address from there. */
const times = timing((line) => {
  const cue = transcript.value?.spans.value[line]
  if (cue) embed.value?.seeks(cue.from)
})

// The transcript moves under what is playing: the line being said is drawn in
// the accent, and following is what brings it back into view.
watch(
  () =>
    [
      transcript.value?.times.value,
      transcript.value?.current.value,
      transcript.value?.following.value,
    ] as const,
  () => {
    const held = transcript.value
    if (!held) return
    times.show({
      times: held.times.value,
      current: held.current.value,
      following: held.following.value && !held.typing.value,
    })
  },
  { immediate: true },
)

/**
 * What the menu offers over this address: each run only where it applies, and
 * the one that takes the transcript away last.
 */
const offered = computed(() => {
  const held = transcript.value
  const written = (held?.cues.value.length ?? 0) > 0
  const working = held?.working.value ?? false
  if (working) return []
  return [
    ...(props.state.can(FETCH) ? [{ id: FETCH, text: words.fetch }] : []),
    ...(props.state.can(DOWNLOAD) ? [{ id: DOWNLOAD, text: words.download }] : []),
    ...(written ? [{ id: DROP, text: words.drop }] : []),
  ]
})

/** Where the menu was asked for, and nothing while it is not open. */
const asking = ref<{ at: Position; from: HTMLElement } | null>(null)

const opens = (event: Event) => {
  const button = event.currentTarget
  if (!(button instanceof HTMLElement)) return
  const box = button.getBoundingClientRect()
  asking.value = { at: { x: box.left, y: box.bottom }, from: button }
}

const chose = (id: string) => {
  asking.value = null
  props.state.asks(id)
}
</script>

<template>
  <div class="link">
    <div class="link__head">
      <LinkEmbed
        ref="embed"
        :address="props.state.address.value!"
        :copy="transcript?.address.value ?? ''"
        :words="words"
      />

      <button
        v-if="offered.length"
        type="button"
        class="link__more"
        :aria-label="words.more"
        :title="words.more"
        aria-haspopup="menu"
        @click="opens"
      >
        <Ellipsis class="link__icon" />
      </button>
    </div>

    <Editor
      v-if="transcript && transcript.cues.value.length > 0"
      class="link__transcript"
      :model-value="transcript.prose.value"
      :readonly="!transcript.editable.value"
      :live="false"
      :extensions="times.extension"
      :aria-label="words.transcript"
      @update:model-value="(body: string) => transcript?.typed(body)"
      @save="transcript?.keep()"
    />

    <p v-else class="link__note">{{ words.nothingFetched }}</p>

    <Menu
      v-if="asking"
      :items="offered"
      :at="asking.at"
      :from="asking.from"
      open
      :name="words.more"
      @choose="chose"
      @close="asking = null"
    />
  </div>
</template>

<style scoped>
.link {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-inline-size: 0;
  min-block-size: 0;
}

/* The player, with what can be asked over the address standing on it. The
   player is as wide as the tab and no wider, so nothing here scrolls across. */
.link__head {
  position: relative;
  flex: none;
  min-inline-size: 0;
  overflow-x: hidden;
}

.link__more {
  position: absolute;
  inset-block-start: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  display: flex;
  padding: calc(var(--numen-inset) / 2);
  border: 0;
  border-radius: var(--numen-radius);
  background: var(--numen-veil);
  color: var(--numen-text);
  cursor: pointer;
}

.link__icon {
  inline-size: 1rem;
  block-size: 1rem;
}

/* The transcript runs the width of the pane, clear of the rule the player
   stands on, and the editor scrolls so the bar stands at the edge. */
.link .link__transcript {
  --editor-lead: var(--numen-inset);
  --editor-margin: var(--numen-gutter);

  flex: 1;
  min-block-size: 0;
  isolation: isolate;
  font-size: inherit;
}

.link__note {
  margin: 0;
  padding: var(--numen-gutter);
  color: var(--numen-hushed);
}
</style>
