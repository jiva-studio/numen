/** The name of an icon in the catalog: `trash-2` is `Trash2`. */
export const getIconKey = (name: string): string =>
  name.replace(/(^|-)([a-z0-9])/g, (_match, _dash, letter: string) => letter.toUpperCase())

/** What the module exports besides icons: the base component and the helpers. */
export const isIconKey = (key: string): boolean =>
  /^[A-Z]/.test(key) && key !== 'Icon' && !key.startsWith('Lucide')
