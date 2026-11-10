/**
 * The icon a plugin's manifest names, by its Lucide name. The same name gives the same component,
 * which draws the icon once the set is loaded, and nothing for a name with no icon.
 */
import type { Component } from 'vue'
import { createNamedIcon } from '@numen/ui'

export const getNamedIcon = (name: string): Component => createNamedIcon(name)
