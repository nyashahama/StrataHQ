import type { Config } from 'tailwindcss'

const config: Config = {
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './hooks/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        page: '#F7F6F3',
        ink: {
          DEFAULT: '#37352F',
          2: '#4D4B45',
        },
        muted: {
          DEFAULT: '#787672',
          2: '#A8A49E',
        },
        border: {
          DEFAULT: '#E3E2DF',
          2: '#CECDCA',
        },
        accent: {
          DEFAULT: '#2B6CB0',
          bg: '#EBF2FA',
          dim: 'rgba(43,108,176,0.08)',
        },
        green: {
          DEFAULT: '#276749',
          bg: '#F0FFF4',
        },
        red: {
          DEFAULT: '#9B2C2C',
          bg: '#FFF5F5',
        },
        yellowbg: '#FFFBEB',
      },
      fontFamily: {
        serif: ['var(--font-lora)', 'Georgia', 'serif'],
        sans: ['var(--font-dm-sans)', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        sm: '6px',
        DEFAULT: '8px',
        lg: '12px',
      },
      boxShadow: {
        sm: '0 1px 3px rgba(55,53,47,0.08), 0 1px 2px rgba(55,53,47,0.04)',
        DEFAULT: '0 4px 16px rgba(55,53,47,0.10), 0 1px 4px rgba(55,53,47,0.06)',
        lg: '0 16px 48px rgba(55,53,47,0.14), 0 4px 12px rgba(55,53,47,0.08)',
      },
      maxWidth: {
        container: '1080px',
      },
    },
  },
  plugins: [],
}

export default config
