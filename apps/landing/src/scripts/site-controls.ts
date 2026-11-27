import { EN } from './i18n-en'
import { RU } from './i18n-ru'
import { SR } from './i18n-sr'
import { readStorageValue, writeStorageValue } from './storage'

const I18N: Readonly<Record<string, Readonly<Record<string, string>>>> = { en: EN, ru: RU, sr: SR }

function toggleTheme(): void {
  const current = document.documentElement.getAttribute('data-theme') || 'light'
  const next = current === 'light' ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', next)
  writeStorageValue('numen-theme', next)
}

function setProductsMenuOpen(isOpen: boolean): void {
  document.getElementById('products-menu')?.toggleAttribute('hidden', !isOpen)
  document.getElementById('products-btn')?.setAttribute('aria-expanded', String(isOpen))
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
    const count = el.dataset.i18nCount
    if (selector === 'text' && count) el.textContent = text.replace('{count}', count)
    else if (selector === 'html') el.innerHTML = text
    else if (selector === 'aria') el.setAttribute('aria-label', text)
    else if (selector === 'placeholder') el.setAttribute('placeholder', text)
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
  translate('placeholder', 'data-i18n-placeholder', dict)
  writeStorageValue('numen-lang', lang)
}

function toggleLanguageMenu(event: Event): void {
  event.stopPropagation()
  setProductsMenuOpen(false)
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

function bindProductsMenu(): void {
  const button = document.getElementById('products-btn')
  const menu = document.getElementById('products-menu')
  if (!button || !menu) return
  button.addEventListener('click', (event) => {
    event.stopPropagation()
    closeLanguageMenu()
    setProductsMenuOpen(menu.hasAttribute('hidden'))
  })
  document.addEventListener('click', (event) => {
    if (event.target instanceof Node && !menu.contains(event.target)) setProductsMenuOpen(false)
  })
  document.addEventListener('keydown', (event) => {
    if (event.key !== 'Escape' || menu.hasAttribute('hidden')) return
    setProductsMenuOpen(false)
    button.focus()
  })
}

/** Wires the theme toggle, the language menu and the products menu, and applies the saved language. */
export function initSiteControls(): void {
  document.getElementById('theme-toggle')?.addEventListener('click', toggleTheme)
  bindLanguageMenu()
  bindProductsMenu()
  const saved = readStorageValue('numen-lang')
  if (saved && I18N[saved]) setLanguage(saved)
}
