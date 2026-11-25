<script setup lang="ts">
/** The row of the card "Synced vaults" before the vaults are read: the same geometry, no text. */
import { Skeleton } from '@numen/ui'
</script>

<template>
  <li class="vault-row vault-row--loading">
    <Skeleton class="vault-row__icon" wide="var(--vault-row-icon)" high="var(--vault-row-icon)" />
    <div class="vault-row__label">
      <Skeleton wide="calc(10 * var(--numen-chrome-unit))" high="1em" />
      <Skeleton wide="calc(15 * var(--numen-chrome-unit))" high="0.875em" />
    </div>
  </li>
</template>
