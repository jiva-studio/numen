<script setup lang="ts">
/**
 * A plex tab: the picture, and the menu on a node of it.
 *
 * Every gesture is handed to what the tab holds. The menu stands on a node of
 * this picture and goes when the picture does.
 */
import { Menu, Plex } from '@numen/ui'
import type { MenuOpening, PlexRelatedSeat, PlexShowing } from '@numen/ui'
import { CREATABLE } from '../creating'
import { ITEMS } from '../menu'
import type { Held } from './kind'

const props = defineProps<{ held: Held }>()
</script>

<template>
  <Plex
    v-if="props.held.picture.value"
    :neighbourhood="props.held.picture.value!"
    :creatable="CREATABLE"
    @activate="(path: string) => props.held.activate(path)"
    @create="(from: string, seat: PlexRelatedSeat) => void props.held.made(from, seat)"
    @link="
      (from: string, to: string, seat: PlexRelatedSeat) => void props.held.joined(from, to, seat)
    "
    @menu="
      (
        path: string,
        at: { x: number; y: number },
        from: SVGGElement,
        opening: MenuOpening,
      ) => props.held.asks({ path, at, from, opening })
    "
    @show="(path: string, how: PlexShowing) => props.held.opens(path, how)"
    @dismiss="props.held.dismiss()"
  />

  <Menu
    v-if="props.held.menu.value"
    :items="ITEMS"
    :at="props.held.menu.value!.at"
    :from="props.held.menu.value!.from"
    :opening="props.held.menu.value!.opening"
    open
    @choose="(id: string) => props.held.chose(id)"
    @dismiss="props.held.dismiss()"
  />
</template>
