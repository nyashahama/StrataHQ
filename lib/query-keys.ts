export const portfolioKeys = {
  overview: () => ["agent", "portfolio", "overview"] as const,
  attention: () => ["agent", "portfolio", "attention"] as const,
}

export const schemeKeys = {
  overview: (schemeId: string) => ["scheme", schemeId, "overview"] as const,
  attention: (schemeId: string) => ["scheme", schemeId, "attention"] as const,
  attentionQueue: (schemeId: string) => ["scheme", schemeId, "attention-queue"] as const,
  members: (schemeId: string) => ["scheme", schemeId, "members"] as const,
  membersUnits: (schemeId: string) => ["scheme", schemeId, "members", "units"] as const,
  communicationsBase: (schemeId: string) => ["scheme", schemeId, "communications"] as const,
  communications: (schemeId: string, type: string) =>
    ["scheme", schemeId, "communications", type] as const,
  documentsBase: (schemeId: string) => ["scheme", schemeId, "documents"] as const,
  documents: (schemeId: string, category: string) =>
    ["scheme", schemeId, "documents", category] as const,
  financialsBase: (schemeId: string) => ["scheme", schemeId, "financials"] as const,
  financials: (schemeId: string, period: string) =>
    ["scheme", schemeId, "financials", period] as const,
  maintenance: (schemeId: string) => ["scheme", schemeId, "maintenance"] as const,
  levy: (schemeId: string) => ["scheme", schemeId, "levy"] as const,
  agm: (schemeId: string) => ["scheme", schemeId, "agm"] as const,
  agmMembers: (schemeId: string) => ["scheme", schemeId, "agm", "members"] as const,
}