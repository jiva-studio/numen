export type Environment = {
  readonly id: string
  readonly label: string
  readonly order: number
}

const BROWSERS: Readonly<Record<string, string>> = {
  chromium: 'Chromium',
  webkit: 'WebKit',
  firefox: 'Firefox',
}

const SYSTEMS: Readonly<Record<string, string>> = {
  linux: 'Linux',
  darwin: 'macOS',
  win32: 'Windows',
}

const getOrder = (names: readonly string[], name: string): number => {
  const index = names.indexOf(name)
  return index === -1 ? names.length : index
}

export const getEnvironment = (project: string, platform: string): Environment => {
  const browsers = Object.keys(BROWSERS)
  const systems = Object.keys(SYSTEMS)
  return {
    id: `${project}-${platform}`,
    label: `${BROWSERS[project] ?? project} · ${SYSTEMS[platform] ?? platform}`,
    order: getOrder(systems, platform) * (browsers.length + 1) + getOrder(browsers, project),
  }
}
