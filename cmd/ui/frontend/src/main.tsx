import './global.css'
import React from 'react'
import {createRoot} from 'react-dom/client'
import {Provider} from '@react-spectrum/s2'
import App from './App'

const container = document.getElementById('root')

const root = createRoot(container!)

// Sem `colorScheme`: o tema é o Sistema (docs/FRONTEND.md, Tema).
root.render(
    <React.StrictMode>
        <Provider locale="pt-BR">
            <App/>
        </Provider>
    </React.StrictMode>
)
