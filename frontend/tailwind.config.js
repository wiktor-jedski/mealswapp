/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{svelte,ts}'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        background: 'var(--color-bg-primary)',
        surface: 'var(--color-bg-surface)',
        primary: 'var(--color-primary)',
        secondary: 'var(--color-secondary)',
        accent: 'var(--color-accent)',
        error: 'var(--color-error)',
        'text-primary': 'var(--color-text-primary)',
        'text-muted': 'var(--color-text-muted)',
        'dark-background': '#0A0F0A',
        'dark-surface': '#161D16',
        'dark-primary': '#4ADE80',
        'dark-secondary': '#86EFAC',
        'dark-accent': '#FFB86C',
        'dark-error': '#F87171',
        'dark-text-primary': '#F3F4F6',
        'dark-text-muted': '#9CA3AF'
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        mono: ['Roboto Mono', 'monospace']
      },
      maxWidth: {
        app: '1280px'
      }
    }
  },
  plugins: []
};
