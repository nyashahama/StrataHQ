import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

const PUBLIC_PATHS = ['/', '/auth/login', '/auth/register', '/auth/forgot-password', '/auth/reset-password', '/early-access']

function isPublicPath(pathname: string): boolean {
  return PUBLIC_PATHS.some(
    (p) => pathname === p || pathname.startsWith('/auth/reset-password/')
  )
}

function readSessionCookie(request: NextRequest) {
  const match = request.cookies.get('sh_session')?.value
  if (!match) return null
  try {
    return JSON.parse(decodeURIComponent(match))
  } catch {
    return null
  }
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  if (isPublicPath(pathname)) {
    const session = readSessionCookie(request)
    if (session && pathname.startsWith('/auth/')) {
      return NextResponse.redirect(new URL('/app', request.url))
    }
    return NextResponse.next()
  }

  const session = readSessionCookie(request)
  if (!session) {
    const loginUrl = new URL('/auth/login', request.url)
    loginUrl.searchParams.set('redirect', pathname)
    return NextResponse.redirect(loginUrl)
  }

  if (pathname.startsWith('/agent')) {
    if (session.role !== 'admin') {
      const schemeId = session.scheme_memberships?.[0]?.scheme_id
      return NextResponse.redirect(
        new URL(schemeId ? `/app/${schemeId}` : '/auth/login', request.url)
      )
    }
  }

  if (pathname.startsWith('/app')) {
    if (session.role === 'admin') {
      return NextResponse.next()
    }
    const schemeId = pathname.split('/')[2]
    const hasAccess = session.scheme_memberships?.some(
      (m: { scheme_id: string }) => m.scheme_id === schemeId
    )
    if (!hasAccess) {
      const primaryScheme = session.scheme_memberships?.[0]?.scheme_id
      return NextResponse.redirect(
        new URL(primaryScheme ? `/app/${primaryScheme}` : '/auth/login', request.url)
      )
    }
  }

  return NextResponse.next()
}

export const config = {
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico).*)'],
}
