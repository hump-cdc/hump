import type { FC } from 'react'

import { type ChartConfig, THEMES } from './chart-context'

type ChartStyleProps = {
    id: string
    config: ChartConfig
}

export const ChartStyle: FC<ChartStyleProps> = ({ id, config }) => {
    const colorConfig = Object.entries(config).filter(([, config]) => config.theme ?? config.color)

    if (!colorConfig.length) {
        return null
    }

    return (
        <style
            // biome-ignore lint/security/noDangerouslySetInnerHtml: required to inject per-chart color CSS variables
            dangerouslySetInnerHTML={{
                __html: Object.entries(THEMES)
                    .map(
                        ([theme, prefix]) => `
${prefix} [data-chart=${id}] {
${colorConfig
    .map(([key, itemConfig]) => {
        const color = itemConfig.theme?.[theme as keyof typeof itemConfig.theme] ?? itemConfig.color
        return color ? `  --color-${key}: ${color};` : null
    })
    .join('\n')}
}
`,
                    )
                    .join('\n'),
            }}
        />
    )
}
