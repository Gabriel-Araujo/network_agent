import './global.css'
import React from 'react'
import {createRoot} from 'react-dom/client'
import {Provider} from '@react-spectrum/s2'
import App from './App'
import {useTheme} from './theme'

const container = document.getElementById('root')

// `background="layer-1"` pinta a moldura uma vez só, sob a grade toda (inclusive a margem do cartão).
// Sistema (e nada salvo) omite o `colorScheme`: o app acompanha o Windows (docs/FRONTEND.md, Tema).
function Root() {
    const theme = useTheme()
    return (
        <Provider locale="pt-BR" colorScheme={theme === 'system' ? undefined : theme} background="layer-1">
            <App/>
        </Provider>
    )
}

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <Root/>
    </React.StrictMode>
)
