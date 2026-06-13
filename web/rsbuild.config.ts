import { tanstackRouter } from '@tanstack/router-plugin/rspack'

import { defineConfig } from '@rsbuild/core'
import { pluginReact } from '@rsbuild/plugin-react'

import { themeInitScript } from './src/shared/theme/theme-script'

export default defineConfig({
    plugins: [pluginReact()],
    html: {
        template: './index.html',
        title: 'hump',
        mountId: 'app',
        meta: { viewport: 'width=device-width, initial-scale=1' },
        tags: [{ tag: 'script', children: themeInitScript, head: true, append: false }],
    },
    source: { entry: { index: './app/main.tsx' } },
    output: {
        distPath: { root: '../internal/web/assets' },
        cleanDistPath: true,
        legalComments: 'none',
    },
    server: { port: 3000 },
    performance: {
        bundleAnalyze: process.env.BUNDLE_ANALYZE ? {} : undefined,
    },
    tools: {
        cssLoader: {
            url: { filter: (url: string) => !url.startsWith('/fonts/') },
        },
        rspack: {
            plugins: [
                tanstackRouter({
                    target: 'react',
                    autoCodeSplitting: true,
                    routesDirectory: './app/routes',
                    generatedRouteTree: './app/routeTree.gen.ts',
                }),
            ],
        },
    },
})
