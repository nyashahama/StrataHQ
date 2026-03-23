'use client'

import React, { createContext, useContext, useEffect, useState } from 'react'

export interface MockUser {
  role: 'agent' | 'trustee' | 'resident'
  orgName: string
  schemeName: string
  schemeId: string
  unitIdentifier?: string
  isWizardComplete: boolean
}

interface MockAuthContextValue {
  user: MockUser | null
  login: (user: MockUser) => void
  logout: () => void
}

const STORAGE_KEY = 'stratahq_mock_user'

const MockAuthContext = createContext<MockAuthContextValue | null>(null)

export function MockAuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<MockUser | null>(null)

  useEffect(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        setUser(JSON.parse(stored))
      }
    } catch {
      // ignore parse errors
    }
  }, [])

  function login(newUser: MockUser) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(newUser))
    setUser(newUser)
  }

  function logout() {
    localStorage.removeItem(STORAGE_KEY)
    setUser(null)
  }

  return (
    <MockAuthContext.Provider value={{ user, login, logout }}>
      {children}
    </MockAuthContext.Provider>
  )
}

export function useMockAuth(): MockAuthContextValue {
  const ctx = useContext(MockAuthContext)
  if (!ctx) {
    throw new Error('useMockAuth must be used within a MockAuthProvider')
  }
  return ctx
}
