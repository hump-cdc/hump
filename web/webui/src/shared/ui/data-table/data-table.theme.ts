import { themeQuartz } from 'ag-grid-community'

export const dataTableTheme = themeQuartz.withParams({
    backgroundColor: 'var(--surface-1)',
    foregroundColor: 'var(--ink)',
    headerBackgroundColor: 'var(--surface-2)',
    headerTextColor: 'var(--ink-subtle)',
    borderColor: 'var(--hairline)',
    rowBorder: { style: 'solid', width: 1, color: 'var(--hairline)' },
    headerRowBorder: { style: 'solid', width: 1, color: 'var(--hairline)' },
    rowHoverColor: 'var(--surface-2)',
    accentColor: 'var(--primary)',
    fontFamily: 'Inter, ui-sans-serif, system-ui, sans-serif',
    fontSize: 13,
    headerFontSize: 11,
    headerFontWeight: 600,
    rowHeight: 44,
    headerHeight: 38,
    cellHorizontalPadding: 14,
    wrapperBorderRadius: 12,
    borderRadius: 0,
})
