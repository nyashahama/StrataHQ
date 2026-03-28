import { NextRequest } from 'next/server'
import Anthropic from '@anthropic-ai/sdk'
import { mockPortfolio, mockScheme, mockUnits } from '@/lib/mock/scheme'
import { mockLevyRoll, mockLevyPeriod, mockCollectionTrend } from '@/lib/mock/levy'
import { mockMaintenanceRequests } from '@/lib/mock/maintenance'
import { mockBudgetLines, mockReserveFund } from '@/lib/mock/financials'
import { mockAgmMeeting, mockAgmResolutions, mockUpcomingAgm, mockUpcomingResolutions } from '@/lib/mock/agm'
import { mockNotices } from '@/lib/mock/communications'
import { mockMembers } from '@/lib/mock/members'

const SCHEME_CONTEXT = JSON.stringify(
  {
    portfolio: mockPortfolio,
    currentScheme: mockScheme,
    units: mockUnits,
    levyPeriod: mockLevyPeriod,
    levyRoll: mockLevyRoll,
    collectionTrend: mockCollectionTrend,
    maintenanceRequests: mockMaintenanceRequests,
    budgetLines: mockBudgetLines,
    reserveFund: mockReserveFund,
    agmMeetings: [mockAgmMeeting, mockUpcomingAgm],
    resolutions: [...mockAgmResolutions, ...mockUpcomingResolutions],
    notices: mockNotices,
    members: mockMembers,
  },
  null,
  2
)

const SYSTEM_PROMPT = `You are StrataHQ Copilot — an intelligent assistant for property managing agents in South Africa.

You help agents manage sectional title schemes (body corporates) by answering questions about levy collections, maintenance jobs, AGM status, financials, compliance, and communications. You also draft professional letters, notices, and reports when asked.

Rules:
- Always be specific. Cite actual names, amounts, percentages, and dates from the data below.
- Be concise. Use bullet points for lists. Answer in 3–6 sentences unless a longer response (like a drafted letter) is explicitly requested.
- For financial amounts, use South African Rand (R) formatting (e.g. R 2 450).
- When drafting letters or notices, produce formal, professional documents ready to send. Include the scheme name, date, and proper salutations.
- If asked about compliance, reference STSMA (Sectional Titles Schemes Management Act) requirements.
- Today's date is ${new Date().toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}.

LIVE PORTFOLIO DATA:
${SCHEME_CONTEXT}`

export async function POST(request: NextRequest) {
  const apiKey = process.env.ANTHROPIC_API_KEY
  if (!apiKey) {
    return new Response(
      'ANTHROPIC_API_KEY is not configured. Add it to your .env.local file to use StrataHQ Copilot.',
      { status: 503, headers: { 'Content-Type': 'text/plain' } }
    )
  }

  const { message, history } = (await request.json()) as {
    message: string
    history: Array<{ role: 'user' | 'assistant'; content: string }>
  }

  const client = new Anthropic({ apiKey })

  const stream = await client.messages.create({
    model: 'claude-haiku-4-5-20251001',
    max_tokens: 1024,
    stream: true,
    system: SYSTEM_PROMPT,
    messages: [...history, { role: 'user', content: message }],
  })

  const encoder = new TextEncoder()
  const readable = new ReadableStream({
    async start(controller) {
      try {
        for await (const event of stream) {
          if (
            event.type === 'content_block_delta' &&
            event.delta.type === 'text_delta'
          ) {
            controller.enqueue(encoder.encode(event.delta.text))
          }
        }
      } finally {
        controller.close()
      }
    },
  })

  return new Response(readable, {
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      'Cache-Control': 'no-cache',
    },
  })
}
