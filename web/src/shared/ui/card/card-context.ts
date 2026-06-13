import { createContext, useContext } from 'react'

type CardContextValue = {
    separated: boolean
}

export const CardContext = createContext<CardContextValue>({ separated: false })

export const useCardContext = (): CardContextValue => useContext(CardContext)
