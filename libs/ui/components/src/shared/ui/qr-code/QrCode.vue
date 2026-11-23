<script setup lang="ts">
/**
 * A text drawn as a QR code of error correction level M, on a white ground with a quiet zone, so a camera reads it
 * in a dark theme too. Nothing is fetched: the squares are computed here.
 */
import { computed } from 'vue'
import { encode } from 'uqr'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<{ text: string; size?: number; label?: string }>(), {
  size: 220,
  label: 'QR code',
})

/* --------------------------------- State ---------------------------------- */
const QUIET_ZONE = 2

const modules = computed(() => encode(props.text, { ecc: 'M', border: 0 }))
const span = computed(() => modules.value.length + 2 * QUIET_ZONE)

/** The dark squares as one path, a row of runs at a time. */
const path = computed(() => {
  const parts: string[] = []
  modules.value.forEach((row, y) => {
    let start = -1
    row.forEach((isDark, x) => {
      if (isDark && start < 0) start = x
      if ((!isDark || x === row.length - 1) && start >= 0) {
        const end = isDark ? x + 1 : x
        parts.push(`M${start + QUIET_ZONE} ${y + QUIET_ZONE}h${end - start}v1h${start - end}z`)
        start = -1
      }
    })
  })
  return parts.join('')
})
</script>

<template>
  <svg
    class="qr-code"
    role="img"
    :aria-label="label"
    :width="size"
    :height="size"
    :viewBox="`0 0 ${span} ${span}`"
    shape-rendering="crispEdges"
  >
    <rect :width="span" :height="span" fill="#ffffff" />
    <path :d="path" fill="#000000" />
  </svg>
</template>
