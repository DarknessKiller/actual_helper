import { mount } from 'svelte'
import App from './App.svelte'
import './app.css'
import { initTheme } from './lib/stores/theme.js'

initTheme()

const app = mount(App, { target: document.getElementById('app')! })
export default app