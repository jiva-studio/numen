<script setup lang="ts">
/**
 * The screen a window holding nothing shows: the vaults this installation
 * holds, and the ways into the one it is showing.
 */
import { WelcomePage } from '@numen/workbench'
import { WORDS as words } from '@/shared/ui/words'
import { VERSION } from '../lib/version'
import { useWelcomeScreen, type WelcomeScreenProps } from '../model/useWelcomeScreen'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<WelcomeScreenProps>()

/* --------------------------------- State ---------------------------------- */
const { adding, ways, vaults, onRun, onOpen, onTakeFooter } = useWelcomeScreen(props)
</script>

<template>
  <WelcomePage
    :ways="ways"
    :vaults="vaults"
    :heading="words.vaults"
    :footer="adding"
    :version="VERSION"
    @run="onRun"
    @open="onOpen"
    @take-footer="onTakeFooter"
  />
</template>
