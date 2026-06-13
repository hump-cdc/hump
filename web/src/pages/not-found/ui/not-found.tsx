import { Link, useLocation } from '@tanstack/react-router'

import type { FC, ReactNode } from 'react'

import { Button } from '@/shared/ui/button'

const truncate = (value: string, max = 48): string => (value.length <= max ? value : `${value.slice(0, max - 1)}…`)

const Prompt: FC<{ children: ReactNode }> = ({ children }) => (
    <div className="flex items-center gap-2">
        <span className="font-semibold text-[#27a644]">➜</span>
        <span className="font-semibold text-[#5e6ad2]">~/hump</span>
        <span className="text-[#f7f8f8]">{children}</span>
    </div>
)

export const NotFound: FC = () => {
    const pathname = useLocation().pathname
    const path = truncate(pathname && pathname !== '/' ? pathname : '/the-page-you-wanted')

    return (
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
                    <span className="flex-1 text-center font-mono text-sm text-[#8a8f98]">bash — hump — 80×24</span>
                    <span className="w-13" />
                </div>

                <div className="flex flex-col gap-3 bg-black p-6 font-mono text-md leading-relaxed">
                    <Prompt>cd {path}</Prompt>
                    <div className="text-[#eb5757]">cd: no such file or directory: {path}</div>

                    <Prompt>grep -r "that page" .</Prompt>
                    <div className="text-[#8a8f98]">0 results. We looked everywhere. Even prod.</div>

                    <Prompt>git blame</Prompt>
                    <div className="text-[#f2c94c]">fatal: it was DNS. it's always DNS.</div>

                    <Prompt>sudo make it-work</Prompt>
                    <div className="text-[#8a8f98]">Permission denied. Nice try.</div>

                    <div className="mt-1.5 flex items-center gap-2">
                        <span className="font-semibold text-[#d0d6e0]">status: 404 — page not found</span>
                        <span className="inline-block h-4.5 w-2 animate-blink bg-primary" />
                    </div>
                </div>
            </div>

            <div className="flex flex-col items-center gap-3.5 text-center">
                <h1 className="text-5xl font-bold text-ink">We grepped. We found nothing.</h1>
                <p className="text-lg text-ink-subtle">Error 404: this page has left the cluster.</p>
                <Button asChild className="mt-2">
                    <Link to="/" replace>
                        Go to Dashboard
                    </Link>
                </Button>
            </div>
        </div>
    )
}
