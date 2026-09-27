import { type ClassValue, clsx } from 'clsx'
import { extendTailwindMerge } from 'tailwind-merge'

const twMerge = extendTailwindMerge({
    extend: {
        theme: { shadow: ['glow'] },
    },
})

export const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs))
