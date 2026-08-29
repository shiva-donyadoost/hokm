/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        table: {
          900: '#0f172a',
          800: '#134e4a',
          700: '#115e59',
        },
      },
    },
  },
  plugins: [],
}
