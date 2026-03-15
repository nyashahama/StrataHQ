export default function QuoteSection() {
  return (
    <section className="bg-ink">
      <div className="max-w-container mx-auto px-container">
        <div className="reveal max-w-[680px] mx-auto text-center py-[clamp(56px,9vh,80px)]">
          <p className="font-serif text-clamp-quote italic font-normal text-page leading-[1.55] mb-7">
            &ldquo;We went from chasing 30% of our levies every month to collecting
            96% on time. The arrears workflow alone saves us two full days of
            admin.&rdquo;
          </p>
          <p className="text-[14px] text-[rgba(247,246,243,0.5)]">
            <strong className="text-[rgba(247,246,243,0.8)] font-medium">Lindiwe Dlamini</strong>
            {' '}— Managing Director, Pinnacle Property Management, Johannesburg
          </p>
        </div>
      </div>
    </section>
  )
}
