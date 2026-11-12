import {
  Bot,
  BookOpen,
  Cpu,
  LayoutPanelLeft,
  Palette,
  Plug,
  Puzzle,
  Settings,
  SunMoon,
  Wrench,
} from '@lucide/vue'
import type { Component } from 'vue'

const ICONS: Readonly<Record<string, Component>> = {
  agents: Bot,
  connections: Plug,
  engines: Cpu,
  readers: BookOpen,
  themes: Palette,
  tools: Wrench,
  views: LayoutPanelLeft,
  system: Settings,
  appearance: SunMoon,
}

/** The icon a row of the category, or of the System and Appearance pages, is drawn with. */
export const getCategoryIcon = (category: string): Component => ICONS[category] ?? Puzzle
