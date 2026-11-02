/** The version a plugin's package.json states. */
export const readPackageVersion = (text) => JSON.parse(text).version
