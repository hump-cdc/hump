import type { FC, PropsWithChildren } from 'react'

export const ErrorPrompt: FC<PropsWithChildren> = ({ children }) => (
    <div className="flex items-center gap-2">
        <span className="font-semibold text-[#27a644]">➜</span>
        <span className="font-semibold text-[#5e6ad2]">~/hump</span>
        <span className="text-[#f7f8f8]">{children}</span>
    </div>
)
