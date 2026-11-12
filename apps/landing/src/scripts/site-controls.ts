import { EN } from './i18n-en'
import { RU } from './i18n-ru'
import { SR } from './i18n-sr'
import { readStored, writeStored } from './storage'

const I18N: Readonly<Record<string, Readonly<Record<string, string>>>> = { en: EN, ru: RU, sr: SR }

const ARROW_STEP: Readonly<Record<string, number>> = {
  ArrowRight: 1,
  ArrowDown: 1,
  ArrowLeft: -1,
  ArrowUp: -1,
}

function toggleTheme(): void {
  const current = document.documentElement.getAttribute('data-theme') || 'light'
  const next = current === 'light' ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', next)
  writeStored('numen-theme', next)
}

function closeLanguageMenu(): void {
  document.getElementById('lang-menu')?.setAttribute('hidden', '')
  document.getElementById('lang-dropdown-btn')?.setAttribute('aria-expanded', 'false')
}

function translate(
  selector: string,
  attribute: string,
  dict: Readonly<Record<string, string>>,
): void {
  for (const el of document.querySelectorAll<HTMLElement>(`[${attribute}]`)) {
    const key = el.getAttribute(attribute)
    const text = key ? dict[key] : undefined
    if (!text) continue
    if (selector === 'html') el.innerHTML = text
    else if (selector === 'aria') el.setAttribute('aria-label', text)
    else el.textContent = text
  }
}

function setLanguage(lang: string): void {
  const dict = I18N[lang]
  if (!dict) return
  document.documentElement.setAttribute('lang', lang)
  const label = document.getElementById('lang-current-label')
  if (label) label.textContent = lang.toUpperCase()
  for (const item of document.querySelectorAll<HTMLElement>('.lang-dropdown__item')) {
    item.setAttribute('aria-selected', String(item.getAttribute('data-lang') === lang))
  }
  closeLanguageMenu()
  translate('text', 'data-i18n', dict)
  translate('html', 'data-i18n-html', dict)
  translate('aria', 'data-i18n-aria', dict)
  writeStored('numen-lang', lang)
}

function selectApplication(app: string): void {
  document.documentElement.setAttribute('data-app', app)
  const isNumen = app === 'numen'
  for (const el of document.querySelectorAll<HTMLElement>('.app-switch-numen')) {
    el.setAttribute('aria-selected', String(isNumen))
    el.setAttribute('tabindex', isNumen ? '0' : '-1')
  }
  for (const el of document.querySelectorAll<HTMLElement>('.app-switch-flashcards')) {
    el.setAttribute('aria-selected', String(!isNumen))
    el.setAttribute('tabindex', !isNumen ? '0' : '-1')
  }
}

function toggleLanguageMenu(event: Event): void {
  event.stopPropagation()
  const menu = document.getElementById('lang-menu')
  if (!menu) return
  const willOpen = menu.hasAttribute('hidden')
  if (willOpen) menu.removeAttribute('hidden')
  else menu.setAttribute('hidden', '')
  document.getElementById('lang-dropdown-btn')?.setAttribute('aria-expanded', String(willOpen))
}

function bindLanguageMenu(): void {
  document.getElementById('lang-dropdown-btn')?.addEventListener('click', toggleLanguageMenu)
  document.addEventListener('click', (event) => {
    const dropdown = document.getElementById('lang-dropdown')
    if (dropdown && event.target instanceof Node && !dropdown.contains(event.target))
      closeLanguageMenu()
  })
  for (const item of document.querySelectorAll<HTMLButtonElement>('.lang-dropdown__item')) {
    item.addEventListener('click', () => {
      const lang = item.getAttribute('data-lang')
      if (lang) setLanguage(lang)
    })
  }
}

function bindApplicationSwitch(): void {
  const buttons = [...document.querySelectorAll<HTMLButtonElement>('[data-select-app]')]
  for (const [idx, btn] of buttons.entries()) {
    btn.addEventListener('click', () => {
      const app = btn.getAttribute('data-select-app')
      if (app) selectApplication(app)
    })
    btn.addEventListener('keydown', (event) => {
      const step = ARROW_STEP[event.key]
      if (step === undefined) return
      event.preventDefault()
      const next = buttons[(idx + step + buttons.length) % buttons.length]
      const app = next?.getAttribute('data-select-app')
      if (next && app) {
        selectApplication(app)
        next.focus()
      }
    })
  }
}

/** Wires the theme toggle, the language menu and the application switch, and applies the saved language. */
export function initSiteControls(): void {
  document.getElementById('theme-toggle')?.addEventListener('click', toggleTheme)
  bindLanguageMenu()
  bindApplicationSwitch()
  const saved = readStored('numen-lang')
  if (saved && I18N[saved]) setLanguage(saved)
}
