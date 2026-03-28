import type { Metadata } from 'next'
import { Lora, DM_Sans } from 'next/font/google'
import './globals.css'
import { MockAuthProvider } from '@/lib/mock-auth'
import { ThemeProvider } from '@/components/ThemeProvider'

const lora = Lora({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  style: ['normal', 'italic'],
  variable: '--font-lora',
  display: 'swap',
})

const dmSans = DM_Sans({
  subsets: ['latin'],
  weight: ['300', '400', '500', '600'],
  style: ['normal', 'italic'],
  variable: '--font-dm-sans',
  display: 'swap',
})

export const metadata: Metadata = {
  title: 'StrataHQ — Body Corporate Management Platform',
  description:
    'One platform for managing agents, trustees and residents. Levy collections, maintenance, communications and AGMs — clear, connected and under control.',
}

// Anti-flash: runs before React hydration to apply saved theme
const themeScript = `
(function(){
  try {
    var t = localStorage.getItem('stratahq-theme');
    if (!t) t = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    if (t === 'dark') document.documentElement.classList.add('dark');
  } catch(e) {}
})();
`

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${lora.variable} ${dmSans.variable}`} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="bg-page text-ink font-sans antialiased leading-relaxed">
        <ThemeProvider>
          <MockAuthProvider>{children}</MockAuthProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
