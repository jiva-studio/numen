<script setup lang="ts">
/**
 * The days of the week as seven chips, each carrying the share of a day's load
 * it takes. Pressing a day offers the shares, and the day carries what was
 * chosen.
 *
 * A chip is a button that offers a menu: it says the day, the share it carries
 * and whether its shares are open. It holds nothing down.
 *
 * The row is one stop on the way round the screen; the arrow keys move along
 * it and the space bar offers the shares. A day is filled in step with what it
 * carries, so the week is read as the work standing on it without opening
 * anything: a full day is full colour, and a day carrying nothing has none.
 */
import { computed, ref, type HTMLAttributes } from 'vue'
import { RovingFocusGroup, RovingFocusItem } from 'reka-ui'
import { cn } from '@/lib/utils'
import Menu from '@/menu/Menu.vue'
import type { Point } from '@/plex/model'
import { shared, shareOn, SHARES, WEEK, WHOLE, type Day, type Shares } from './week'

// The row and the shares it offers are two things drawn, so what a caller
// names the row by is put on the row itself.
defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The days, in the order they are drawn. */
    days?: readonly Day[]
    /** The shares offered, in the order they are offered. */
    shares?: readonly number[]
    disabled?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { days: () => WEEK, shares: () => SHARES, disabled: false },
)

/** What each day carries. A day not named carries the whole of it. */
const model = defineModel<Shares>({ default: () => ({}) })

/** Which day is being given a share, where its chip stands, and which chip. */
const asking = ref<{ day: string; at: Point; from: HTMLElement } | null>(null)

const offered = computed(() =>
  props.shares.map((share) => ({ id: `${share}`, text: `${share}%` })),
)

/** The share the day being asked about carries, which the menu opens on. */
const carrying = computed(() =>
  asking.value ? `${shareOn(model.value, asking.value.day)}` : null,
)

const asks = (day: string, event: Event) => {
  if (props.disabled) return
  const chip = event.currentTarget
  if (!(chip instanceof HTMLElement)) return
  const box = chip.getBoundingClientRect()
  asking.value = { day, at: { x: box.left, y: box.bottom }, from: chip }
}

const chose = (said: string) => {
  const day = asking.value?.day
  const share = Number(said)
  if (day === undefined || Number.isNaN(share)) return
  model.value = shared(model.value, day, share)
}

/**
 * How strongly a day is filled: the whole of the accent at a day carrying the
 * whole of a day, and no colour at all at a day carrying none. The week is
 * read as the work standing on it.
 */
const filling = (day: string) => {
  const weight = Math.min(Math.max(shareOn(model.value, day), 0), WHOLE)
  return {
    background: `color-mix(in oklab, var(--numen-node-bg), var(--numen-focus-bg) ${weight}%)`,
    color: weight > 50 ? 'var(--numen-focus-fg)' : 'var(--numen-node-fg)',
  }
}
</script>

<template>
  <RovingFocusGroup
    data-slot="days"
    role="toolbar"
    orientation="horizontal"
    v-bind="$attrs"
    :loop="true"
    :class="cn('inline-flex items-center gap-1', props.class)"
  >
    <RovingFocusItem v-for="day in days" :key="day.id" as-child>
      <button
        type="button"
        :disabled="disabled"
        :aria-label="`${day.long}, ${shareOn(model, day.id)}%`"
        aria-haspopup="menu"
        :aria-expanded="asking?.day === day.id"
        :style="filling(day.id)"
        :class="
          cn(
            'inline-flex size-7 shrink-0 items-center justify-center rounded-pill',
            'border border-rule font-sans text-base font-medium',
            'cursor-pointer transition-[background-color,color] duration-100 ease-numen',
            'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
            'disabled:cursor-not-allowed disabled:opacity-50',
          )
        "
        @click="asks(day.id, $event)"
      >
        {{ day.short }}
      </button>
    </RovingFocusItem>
  </RovingFocusGroup>

  <Menu
    v-if="asking"
    :items="offered"
    :at="asking.at"
    :from="asking.from"
    :current="carrying"
    open
    opening="keyboard"
    @choose="chose"
    @dismiss="asking = null"
  />
</template>
