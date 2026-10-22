<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { computed } from 'vue'
import { isNoteAddress } from '@/shared/lib/address'
import { render } from './render'

const props = withDefaults(
  defineProps<{
    /** Markdown, whole or as far as it has arrived. */
    text: string
    /** Still being written: a word that has just arrived is shown arriving. */
    isArriving?: boolean
    /**
     * The addresses this text points at that reach nothing. A link carrying
     * one is drawn as not resolving.
     */
    unresolved?: readonly string[]
  }>(),
  { isArriving: false, unresolved: () => [] },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /**
   * A link in the prose was pressed, with what it points at and the press
   * itself. Nothing here follows it: what a link means is the caller's, and so
   * is whether the browser should go there.
   */
  (event: 'follow', href: string, press: MouseEvent): void
}>()

/* --------------------------------- State ---------------------------------- */
/** The same set for as long as the unresolved addresses have the same members. */
const deadAddresses = computed<ReadonlySet<string>>((previous) => {
  const members = new Set(props.unresolved)
  return previous && isSameSet(previous, members) ? previous : members
})
const renderedContent = computed(() => render(props.text, deadAddresses.value))
const Drawn = () => renderedContent.value

/* -------------------------------- Handlers -------------------------------- */
const onClick = (press: MouseEvent) => {
  const link = (press.target as HTMLElement | null)?.closest?.('a')
  const href = link?.getAttribute('href')
  if (!href) return
  // A note is addressed and not located, so a browser has nowhere to take one.
  if (isNoteAddress(href)) press.preventDefault()
  emit('follow', href, press)
}

/* -------------------------------- Helpers --------------------------------- */
function isSameSet(a: ReadonlySet<string>, b: ReadonlySet<string>) {
  return a.size === b.size && [...a].every((one) => b.has(one))
}
</script>

<template>
  <div
    class="prose prose-sm prose-numen numen max-w-none break-words"
    :class="{ 'prose--arriving': isArriving }"
    @click="onClick"
  >
    <Drawn />
  </div>
</template>

<style scoped>
.prose {
  user-select: text;
  -webkit-user-select: text;
}
</style>
