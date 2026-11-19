/** Whether an address names somewhere this page is not served from. */
export const isExternalLink = (target: URL, here: URL): boolean =>
  target.protocol !== here.protocol || target.host !== here.host
