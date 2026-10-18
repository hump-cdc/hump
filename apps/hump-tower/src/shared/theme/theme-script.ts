import { DARK_CLASS, DARK_MEDIA_QUERY, DEFAULT_THEME, THEME_STORAGE_KEY } from './constants'

/**
 * Synchronous, blocking script injected into <head> (via rsbuild `html.tags`) so it runs
 * BEFORE first paint. It sets the dark class on <html> from the persisted/system theme,
 * eliminating the flash of the wrong theme that would otherwise occur while the deferred app
 * bundle downloads and executes.
 *
 * It runs before any module loads, so the values from `constants.ts` are inlined as literals
 * when this string is built; only the list of theme names is repeated by hand.
 */
export const themeInitScript = `(function () {
  try {
    var stored = localStorage.getItem(${JSON.stringify(THEME_STORAGE_KEY)})
    var theme = stored === 'light' || stored === 'dark' || stored === 'system' ? stored : ${JSON.stringify(DEFAULT_THEME)}
    var isDark = theme === 'dark' || (theme === 'system' && matchMedia(${JSON.stringify(DARK_MEDIA_QUERY)}).matches)
    document.documentElement.classList.toggle(${JSON.stringify(DARK_CLASS)}, isDark)
  } catch (e) {}
})()`
