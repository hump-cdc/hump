import { Link, useLocation } from '@tanstack/react-router'

import type { FC } from 'react'

import { Button } from '@/shared/ui/button'
import { ErrorLine, ErrorPrompt, ErrorTerminal } from '@/shared/ui/error'

const truncate = (value: string, max = 48): string => (value.length <= max ? value : `${value.slice(0, max - 1)}…`)

export const NotFound: FC = () => {
    const pathname = useLocation().pathname
    const path = truncate(pathname && pathname !== '/' ? pathname : '/the-page-you-wanted')

    return (
        <ErrorTerminal
            status="404 — page not found"
            title="We grepped. We found nothing."
            description="Error 404: this page has left the cluster."
            action={
                <Button asChild>
                    <Link to="/" replace>
                        Go to Dashboard
                    </Link>
                </Button>
            }
        >
            <ErrorPrompt>cd {path}</ErrorPrompt>
            <ErrorLine tone="danger">cd: no such file or directory: {path}</ErrorLine>

            <ErrorPrompt>grep -r "that page" .</ErrorPrompt>
            <ErrorLine tone="muted">0 results. We looked everywhere. Even prod.</ErrorLine>

            <ErrorPrompt>git blame</ErrorPrompt>
            <ErrorLine tone="warning">fatal: it was DNS. it&apos;s always DNS.</ErrorLine>

            <ErrorPrompt>sudo make it-work</ErrorPrompt>
            <ErrorLine tone="muted">Permission denied. Nice try.</ErrorLine>
        </ErrorTerminal>
    )
}
