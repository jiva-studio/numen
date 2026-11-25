<script setup lang="ts">
/** The row of the card "Synced vaults" before the vaults are read: the same geometry, no text. */
import { Skeleton } from '@numen/ui'
</script>

<template>
  <li class="vault-row vault-row--loading">
    <span class="vault-row__icon vault-row__icon--loading">
      <Skeleton wide="100%" high="100%" />
    </span>
    <div class="vault-row__label">
      <Skeleton wide="calc(10 * var(--numen-chrome-unit))" high="var(--vault-row-name-height)" />
      <Skeleton wide="calc(15 * var(--numen-chrome-unit))" high="var(--vault-row-line-height)" />
    </div>
  </li>
</template>
