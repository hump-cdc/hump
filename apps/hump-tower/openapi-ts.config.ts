import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
    input: './src/shared/api/openapi.json',
    output: {
        path: './src/shared/api/generated',
    },
    plugins: [
        {
            name: '@hey-api/client-fetch',
            runtimeConfigPath: './src/shared/api/client.ts',
        },
        {
            name: '@hey-api/typescript',
            definitions: { name: '{{name}}Model' },
        },
        {
            name: '@hey-api/sdk',
            transformer: true,
            operations: {
                strategy: 'byTags',
                methods: 'static',
                nesting: (op) => {
                    const parts = op.operationId?.split(/[./]/) ?? []
                    return parts.length > 1 ? parts.slice(1) : [op.method]
                },
            },
        },
        {
            name: '@tanstack/react-query',
            queryKeys: true,
            queryOptions: true,
            infiniteQueryOptions: true,
            mutationKeys: true,
            mutationOptions: true,
        },
        {
            name: 'zod',
            requests: true,
            responses: true,
            metadata: true,
            dates: {
                offset: true,
            },
            definitions: {
                case: 'PascalCase',
                name: '{{name}}Schema',
                types: {
                    infer: {
                        enabled: true,
                        case: 'PascalCase',
                        name: '{{name}}',
                    },
                },
            },
        },
    ],
})
