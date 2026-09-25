import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import Hero from './Hero'
import StatsBar from './StatsBar'
import FeaturesSection from './FeaturesSection'
import InsightsSection from './InsightsSection'
import QuoteSection from './QuoteSection'
import PricingSection from './PricingSection'
import CTASection from './CTASection'
import Footer from './Footer'
import ProblemSection from './ProblemSection'
import ModulesSection from './ModulesSection'
import RolesSection from './RolesSection'

describe('public landing credibility', () => {
  it('identifies examples and avoids unverified adoption, testimonial, and compliance claims', () => {
    vi.stubGlobal('IntersectionObserver', class {
      observe() {}
      unobserve() {}
      disconnect() {}
    })
    const { container } = render(<>
      <Hero />
      <StatsBar />
      <ProblemSection />
      <FeaturesSection />
      <InsightsSection />
      <ModulesSection />
      <RolesSection />
      <QuoteSection />
      <PricingSection />
      <CTASection />
      <Footer />
    </>)
    const content = container.textContent ?? ''

    expect(content).toMatch(/seeded demo|illustrative example/i)
    for (const claim of [
      /2,400\+ schemes/i,
      /180K residents/i,
      /real results from real schemes/i,
      /we went from chasing 30%/i,
      /STSMA compliant|compliant with STSMA/i,
      /STSMA-compliant/i,
      /auto-generated signed minutes/i,
      /accounting integrations/i,
      /most popular/i,
      /limited spots/i,
      /limited number of schemes/i,
      /learns from payment patterns/i,
      /PayFast integration/i,
      /dedicated account manager/i,
      /35 hours\/month of recoverable admin/i,
    ]) {
      expect(content).not.toMatch(claim)
    }
  })
})
