import type { FC, PropsWithChildren, ReactNode } from 'react'

type ErrorTerminalProps = PropsWithChildren<{
    status: ReactNode
    title: ReactNode
    description: ReactNode
    action?: ReactNode
    terminalTitle?: ReactNode
}>

export const ErrorTerminal: FC<ErrorTerminalProps> = ({
    status,
    title,
    description,
    action,
    terminalTitle = 'bash — hump — 80×24',
    children,
}) => (
    <div
        className="flex min-h-screen flex-col items-center justify-center gap-8 bg-canvas p-6"
        style={{
            backgroundImage: 'radial-gradient(var(--hairline-strong) 1px, transparent 1px)',
            backgroundSize: '22px 22px',
        }}
    >
        <div className="w-full max-w-190 overflow-hidden rounded-xl border border-[#23252a] bg-[#0f1011] shadow-[0_24px_60px_-12px_rgba(0,0,0,0.4)]">
            <div className="flex h-10 items-center border-b border-[#23252a] bg-[#141516] px-3.5">
                <div className="flex items-center gap-2">
                    <span className="size-3 rounded-full bg-[#eb5757]" />
                    <span className="size-3 rounded-full bg-[#f2c94c]" />
                    <span className="size-3 rounded-full bg-[#27a644]" />
                </div>
                <span className="flex-1 text-center font-mono text-sm text-[#8a8f98]">{terminalTitle}</span>
                <span className="w-13" />
            </div>

            <div className="flex flex-col gap-3 bg-black p-6 font-mono text-md leading-relaxed">
                {children}

                <div className="mt-1.5 flex items-center gap-2">
                    <span className="font-semibold text-[#d0d6e0]">status: {status}</span>
                    <span className="inline-block h-4.5 w-2 animate-blink bg-primary" />
                </div>
            </div>
        </div>

        <div className="flex flex-col items-center gap-3.5 text-center">
            <h1 className="text-5xl font-bold text-ink">{title}</h1>
            <p className="text-lg text-ink-subtle">{description}</p>
            {action ? <div className="mt-2">{action}</div> : null}
        </div>
    </div>
)
