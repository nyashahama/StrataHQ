import Nav from '@/components/Nav'
import Hero from '@/components/Hero'
import StatsBar from '@/components/StatsBar'
import ProblemSection from '@/components/ProblemSection'
import FeaturesSection from '@/components/FeaturesSection'
import InsightsSection from '@/components/InsightsSection'
import ModulesSection from '@/components/ModulesSection'
import RolesSection from '@/components/RolesSection'
import DemoTeaser from '@/components/DemoTeaser'
import QuoteSection from '@/components/QuoteSection'
import PricingSection from '@/components/PricingSection'
import CTASection from '@/components/CTASection'
import Footer from '@/components/Footer'

export default function Home() {
  return (
    <div className="landing">
      <Nav />
      <main>
        <Hero />
        <StatsBar />
        <ProblemSection />
        <FeaturesSection />
        <InsightsSection />
        <ModulesSection />
        <RolesSection />
        <DemoTeaser />
        <QuoteSection />
        <PricingSection />
        <CTASection />
      </main>
      <Footer />
    </div>
  )
}
