import { extendTailwindMerge } from 'tailwind-merge'
import { type ClassValue, clsx } from 'clsx'

const twMerge = extendTailwindMerge({
    extend: {
        theme: { shadow: ['glow'] },
    },
})

export const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));
