import "./app.css";
import App from "./App.svelte";
import { mount } from "svelte";

// Implements DESIGN-001 SearchView SPA mount point.
const app = mount(App, {
  target: document.getElementById("app") as HTMLElement
});

export default app;
