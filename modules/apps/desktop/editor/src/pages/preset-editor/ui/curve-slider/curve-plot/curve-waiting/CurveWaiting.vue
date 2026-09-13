<script setup lang="ts">
/**
 * What stands in the curve's room while the answer that draws it is read.
 */
import { Spinner } from '@numen/ui'
import { WORDS as words } from '../../../../words'
import '../../curve-slider.css'
</script>

<template>
  <div class="curve-slider__waiting" data-control="waiting" role="status">
    <Spinner class="curve-slider__ring" />
    <span>{{ words.waiting }}</span>
  </div>
</template>
