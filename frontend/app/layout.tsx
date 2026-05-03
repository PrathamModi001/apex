import type { Metadata } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import './globals.css'
import { QueryProvider } from '@/components/query-provider'
import { NavSidebar } from '@/components/nav-sidebar'

const geistSans = Geist({
  variable: '--font-geist-sans',
  subsets: ['latin'],
})

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin'],
})

export const metadata: Metadata = {
  title: 'APEX — Accounts Payable Agent',
  description: 'Autonomous accounts-payable processing agent',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased dark`}
    >
      <body className="min-h-full flex bg-background text-foreground">
        <QueryProvider>
          <NavSidebar />
          <main className="flex-1 ml-56 min-h-screen overflow-y-auto">{children}</main>
        </QueryProvider>
      </body>
    </html>
  )
}
