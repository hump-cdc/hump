import type { Theme } from './types'

/** localStorage key holding the user's theme choice. */
export const THEME_STORAGE_KEY = 'theme'

/** Theme applied when nothing is persisted yet. */
export const DEFAULT_THEME: Theme = 'dark'

/** Class toggled on <html> to activate the dark token set. */
export const DARK_CLASS = 'dark'

/** Class present on <html> only while colors animate during a switch. */
export const THEME_TRANSITION_CLASS = 'theme-transition'

/** How long the color transition runs before the transition class is removed. */
export const THEME_TRANSITION_MS = 250

/** Media query used to resolve the `system` theme. */
export const DARK_MEDIA_QUERY = '(prefers-color-scheme: dark)'
