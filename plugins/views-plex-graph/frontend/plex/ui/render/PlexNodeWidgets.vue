<script setup lang="ts">
/** The views of one side of a node, drawn at the node in the order of the manifest, each given the path of the note. */
import type { Component } from 'vue'
import type { PlacedNode } from '../../lib/node'

defineProps<{
  node: PlacedNode
  side: 'above' | 'below'
  entries: readonly { readonly key: string; readonly component: Component }[]
  path: string
}>()
</script>

<template>
  <g />
</template>
