import type { ThemeRegistrationRaw } from 'shiki/core'

export const humpCode: ThemeRegistrationRaw = {
    name: 'hump-code',
    type: 'light',
    bg: 'var(--syntax-bg)',
    fg: 'var(--syntax-fg)',
    settings: [
        { settings: { background: 'var(--syntax-bg)', foreground: 'var(--syntax-fg)' } },
        {
            scope: ['entity.name.tag', 'meta.tag', 'support.type.property-name'],
            settings: { foreground: 'var(--syntax-tag)' },
        },
        { scope: ['string'], settings: { foreground: 'var(--syntax-string)' } },
        { scope: ['constant.numeric'], settings: { foreground: 'var(--syntax-number)' } },
        { scope: ['constant.language', 'keyword'], settings: { foreground: 'var(--syntax-keyword)' } },
        { scope: ['comment'], settings: { foreground: 'var(--syntax-comment)', fontStyle: 'italic' } },
        {
            scope: ['punctuation', 'keyword.operator', 'entity.name.tag.anchor', 'storage.type'],
            settings: { foreground: 'var(--syntax-punctuation)' },
        },
    ],
}
