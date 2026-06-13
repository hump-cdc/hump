import type { StorybookConfig } from 'storybook-react-rsbuild';

const config: StorybookConfig = {
    stories: ['../src/shared/ui/**/*.mdx', '../src/shared/ui/**/*.stories.@(ts|tsx)'],
    addons: ['@storybook/addon-docs'],
    framework: {
        name: 'storybook-react-rsbuild',
        options: {},
    },
    staticDirs: ['../public'],
    features: {
        backgrounds: false,
        outline: false,
        sidebarOnboardingChecklist: false,
    },
    typescript: { reactDocgen: 'react-docgen' },
    rsbuildFinal: (config) => {
        config.tools ??= {};
        config.tools.cssLoader = {
            url: { filter: (url: string) => !url.startsWith('/fonts/') },
        };
        return config;
    },
};

export default config;
