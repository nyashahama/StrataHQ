'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useMockAuth } from '@/lib/mock-auth'
import SetupWizard from '@/components/wizard/SetupWizard'

export default function SetupPage() {
  const { user } = useMockAuth()
  const router = useRouter()

  useEffect(() => {
    if (user?.isWizardComplete) router.replace('/agent')
  }, [user, router])

  if (user?.isWizardComplete) return null

  return <SetupWizard />
}
