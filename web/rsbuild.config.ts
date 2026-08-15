import { tanstackRouter } from '@tanstack/router-plugin/rspack'

import { readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineConfig, type RsbuildPlugin } from '@rsbuild/core'
import { pluginReact } from '@rsbuild/plugin-react'

import { themeInitScript } from './src/shared/theme/theme-script'

const appDir = fileURLToPath(new URL('./app', import.meta.url))

const routeManifestPlugin: RsbuildPlugin = {
    name: 'route-manifest',
    setup(api) {
        api.onAfterBuild(() => {
            const gen = readFileSync(join(appDir, 'routeTree.gen.ts'), 'utf8')
            const start = gen.indexOf('fullPaths:')
            const end = gen.indexOf('fileRoutesByTo', start)
            const block = start >= 0 && end > start ? gen.slice(start, end) : ''
            const routes = [...new Set([...block.matchAll(/'([^']+)'/g)].map((m) => m[1]))].sort()
            writeFileSync(join(api.context.distPath, 'routes.json'), JSON.stringify(routes))
        })
    },
}

export default defineConfig({
    plugins: [pluginReact(), routeManifestPlugin],
    html: {
        template: './index.html',
        title: 'hump',
        mountId: 'app',
        meta: { viewport: 'width=device-width, initial-scale=1' },
        tags: [{ tag: 'script', children: themeInitScript, head: true, append: false }],
    },
    source: {
        entry: { index: './app/main.tsx' },
        define: {
            VERSION: JSON.stringify(process.env.VERSION ?? '0.0.0'),
            API_URL: JSON.stringify(process.env.API_URL ?? ''),
        },
    },
    output: {
        distPath: { root: '../internal/controller/webui' },
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
