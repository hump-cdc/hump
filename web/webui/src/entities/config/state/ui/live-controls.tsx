import type { FC } from 'react'

import { Play, Square } from 'lucide-react'
import { useShallow } from 'zustand/react/shallow'

import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from '@/shared/ui/dropdown-menu'
import { Tabs, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { useLiveStream } from '@/shared/workers/live-stream'

import { CHART_STEPS, type ChartStep, useLiveControlsStore } from '../model/store'

export const LiveControls: FC = () => {
    const { chartStep, setChartStep } = useLiveControlsStore(
        useShallow((state) => ({
            chartStep: state.chartStep,
            setChartStep: state.setChartStep,
        })),
    )

    // Read from the workers rather than from a flag of our own, so the button and the badge
    // describe what the stream workers are doing even when something else stopped them.
    const { running, toggle } = useLiveStream()

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Badge variant={running ? 'healthy' : 'neutral'} dot className="cursor-pointer">
                    {running ? 'live' : 'paused'}
                </Badge>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="flex w-56 flex-col gap-3 p-3">
                <Button variant={running ? 'secondary' : 'primary'} className="w-full" onClick={toggle}>
                    {running ? <Square /> : <Play />}
                    {running ? 'Stop' : 'Start'}
                </Button>

                <div className="h-px w-full bg-hairline" />

                <div className="flex flex-col gap-2">
                    <span className="text-xs font-semibold tracking-wide text-ink-subtle uppercase">Chart step</span>
                    <Tabs value={chartStep} onValueChange={(value) => setChartStep(value as ChartStep)}>
                        <TabsList className="w-full">
                            {CHART_STEPS.map((option) => (
                                <TabsTrigger key={option} value={option} className="flex-1 py-1 text-xs">
                                    {option}
                                </TabsTrigger>
                            ))}
                        </TabsList>
                    </Tabs>
                </div>
            </DropdownMenuContent>
        </DropdownMenu>
    )
}
