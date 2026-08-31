<script setup lang="ts">
/**
 * The days of the week as seven chips, each carrying the share of a day's load
 * it takes. Pressing a day offers the shares, and the day carries what was
 * chosen.
 *
 * The row is one stop on the way round the screen; the arrow keys move along
 * it and the space bar offers the shares. A day at the whole of it is drawn
 * plain, and the further a day stands under that the stronger it is filled, so
 * the week is read without opening anything.
 */
import { computed, ref, type HTMLAttributes } from 'vue'
import { ToggleGroupItem, ToggleGroupRoot } from 'reka-ui'
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

/** Which day is being given a share, and where its chip stands. */
const asking = ref<{ day: string; at: Point } | null>(null)

const offered = computed(() =>
  props.shares.map((share) => ({ id: `${share}`, text: `${share}%` })),
)

const asks = (day: string, event: Event) => {
  if (props.disabled) return
  const chip = event.currentTarget
  if (!(chip instanceof HTMLElement)) return
  const box = chip.getBoundingClientRect()
  asking.value = { day, at: { x: box.left, y: box.bottom } }
}

const chose = (said: string) => {
  const day = asking.value?.day
  const share = Number(said)
  if (day === undefined || Number.isNaN(share)) return
  model.value = shared(model.value, day, share)
}

/**
 * How strongly a day is filled: nothing at the whole of a day, and the whole
 * of the accent at a day that schedules none of it.
 */
const filling = (day: string) => {
  const share = shareOn(model.value, day)
  const weight = Math.min(Math.max(WHOLE - share, 0), WHOLE)
  return {
    background: `color-mix(in oklab, var(--numen-node-bg), var(--numen-focus-bg) ${weight}%)`,
    color: weight > 50 ? 'var(--numen-focus-fg)' : 'var(--numen-node-fg)',
  }
}
</script>

<template>
  <ToggleGroupRoot
    type="multiple"
    data-slot="days"
    orientation="horizontal"
    v-bind="$attrs"
    :model-value="[]"
    :disabled="disabled"
    :loop="true"
    :class="cn('inline-flex items-center gap-1', props.class)"
  >
    <ToggleGroupItem
      v-for="day in days"
      :key="day.id"
      :value="day.id"
      :aria-label="`${day.long}, ${shareOn(model, day.id)}%`"
      aria-haspopup="menu"
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
    </ToggleGroupItem>
  </ToggleGroupRoot>

  <Menu
    v-if="asking"
    :items="offered"
    :at="asking.at"
    open
    opening="keyboard"
    @choose="chose"
    @dismiss="asking = null"
  />
</template>
