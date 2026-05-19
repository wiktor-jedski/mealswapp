/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{svelte,ts}'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        background: '#F7FCF7',
        surface: '#FFFFFF',
        primary: '#166534',
        secondary: '#DCFCE7',
        accent: '#F97316',
        error: '#DC2626',
        'text-primary': '#111827',
        'text-muted': '#6B7280',
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
