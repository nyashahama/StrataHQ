export interface BillingSubscription {
  customer_id?: string | null;
  subscription_id?: string | null;
  checkout_session_id?: string | null;
  current_period_end?: string | null;
  org_id: string;
  provider: string;
  status: string;
  plan_code: string;
  cancel_at_period_end: boolean;
  entitlement_active: boolean;
  has_portal_access: boolean;
}

export interface BillingCheckoutSession {
  url: string;
  session_id: string;
}

export interface BillingPortalSession {
  url: string;
}
