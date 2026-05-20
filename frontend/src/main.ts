import './app.css';
import { mount } from 'svelte';
import App from './App.svelte';
import { registerServiceWorker } from './lib/offline/serviceWorker';

const app = mount(App, {
  target: document.getElementById('app') as HTMLElement
});

export default app;

if (import.meta.env.PROD) {
  void registerServiceWorker();
}
