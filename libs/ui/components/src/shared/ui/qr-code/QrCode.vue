<script setup lang="ts">
/**
 * A text drawn as a QR code of error correction level M, on a white ground with a quiet zone, so a camera reads it
 * in a dark theme too. Nothing is fetched: the squares are computed here.
 */
import { computed } from 'vue'
import { encode } from 'uqr'
import { getQrPath } from '@/shared/lib/qr/qrPath'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<{ text: string; size?: number; label?: string }>(), {
  size: 220,
  label: 'QR code',
})

/* --------------------------------- State ---------------------------------- */
const QUIET_ZONE = 2

const modules = computed(() => encode(props.text, { ecc: 'M', border: 0 }).data)
const span = computed(() => modules.value.length + 2 * QUIET_ZONE)

/** The dark squares as one path. */
const path = computed(() => getQrPath(modules.value, QUIET_ZONE))
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
