import React from 'react';
import type { Decorator, Preview } from '@storybook/react';
import { ThemeProvider } from '../src/shared/theme'
import { TooltipProvider } from '../src/shared/ui/tooltip'

import '../app/styles.css';

type ThemeMode = 'light' | 'dark' | 'side-by-side';

const panelClass = (padding: string) =>
    `${padding} flex flex-1 items-center justify-center bg-canvas text-ink`;

const Panel = ({
    padding,
    dark = false,
    children,
}: {
    padding: string;
    dark?: boolean;
    children: React.ReactNode;
}) => (
    <div className={`${dark ? 'dark ' : ''}${panelClass(padding)}`}>
        <ThemeProvider>
            <TooltipProvider>{children}</TooltipProvider>
        </ThemeProvider>
    </div>
);

const withTheme: Decorator = (Story, context) => {
    const disableSideBySide = context.parameters.disableSideBySide === true;
    const disablePadding = context.parameters.disablePadding === true;
    const raw = (context.globals.theme as ThemeMode) || 'side-by-side';
    const theme = disableSideBySide && raw === 'side-by-side' ? 'light' : raw;
    const padding = disablePadding ? 'p-0' : 'p-4';

    if (theme === 'side-by-side') {
        return (
            <div className="flex min-h-screen flex-row flex-nowrap">
                <Panel padding={padding}>
                    <Story />
                </Panel>
                <Panel padding={padding} dark>
                    <Story />
                </Panel>
            </div>
        );
    }

    return (
        <div className="flex min-h-screen flex-row flex-nowrap">
            <Panel padding={padding} dark={theme === 'dark'}>
                <Story />
            </Panel>
        </div>
    );
};

const preview: Preview = {
    globalTypes: {
        theme: {
            description: 'Theme',
            toolbar: {
                title: 'Theme',
                icon: 'circlehollow',
                items: [
                    { value: 'light', icon: 'sun', title: 'Light' },
                    { value: 'dark', icon: 'moon', title: 'Dark' },
                    { value: 'side-by-side', icon: 'sidebar', title: 'Side by Side' },
                ],
                dynamicTitle: true,
            },
        },
    },
    initialGlobals: {
        theme: 'side-by-side',
    },
    decorators: [withTheme],
    parameters: {
        layout: 'fullscreen',
        controls: { matchers: { color: /(background|color)$/i, date: /Date$/i } },
        interactions: { disable: true },
        docs: { codePanel: true },
    },
};

export default preview;
