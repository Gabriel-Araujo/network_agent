import './global.css'
import React from 'react'
import {createRoot} from 'react-dom/client'
import {Provider} from '@react-spectrum/s2'
import App from './App'
import {useTheme} from './theme'

const container = document.getElementById('root')

// Sistema (e nada salvo) omite o `colorScheme`: o app acompanha o Windows (docs/FRONTEND.md, Tema).
function Root() {
    const theme = useTheme()
    return (
        <Provider locale="pt-BR" colorScheme={theme === 'system' ? undefined : theme}>
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
