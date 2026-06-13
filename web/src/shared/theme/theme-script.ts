/**
 * Synchronous, blocking script injected into <head> (via rsbuild `html.tags`) so it runs
 * BEFORE first paint. It sets the `.dark` class on <html> from the persisted/system theme,
 * eliminating the flash of the wrong theme that would otherwise occur while the deferred app
 * bundle downloads and executes.
 *
 * This is a plain string with no imports — it runs before any module loads, so the literals
 * below MUST stay in sync with `constants.ts` (THEME_STORAGE_KEY, DEFAULT_THEME, DARK_CLASS,
 * DARK_MEDIA_QUERY).
 */
export const themeInitScript = `(function () {
  try {
    var stored = localStorage.getItem('theme')
    var theme = stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'dark'
    var isDark = theme === 'dark' || (theme === 'system' && matchMedia('(prefers-color-scheme: dark)').matches)
    document.documentElement.classList.toggle('dark', isDark)
  } catch (e) {}
})()`
