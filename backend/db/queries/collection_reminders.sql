-- name: GetCollectionReminderContext :one
SELECT
    la.id AS levy_account_id,
    lp.scheme_id,
    s.name AS scheme_name,
    u.id AS unit_id,
    u.identifier AS unit_identifier,
    u.owner_name,
    (la.amount_cents - la.paid_cents) AS outstanding_cents,
    (CURRENT_DATE - la.due_date) AS days_overdue,
    COALESCE(owner_user.email, '') AS owner_email,
    COALESCE(thread.phone_number, '') AS whatsapp_phone,
    COALESCE(thread.connected, false) AS whatsapp_connected
FROM levy_accounts la
JOIN levy_periods lp ON lp.id = la.period_id
JOIN schemes s ON s.id = lp.scheme_id
JOIN units u ON u.id = la.unit_id
LEFT JOIN scheme_memberships owner_membership
    ON owner_membership.scheme_id = s.id
   AND owner_membership.unit_id = u.id
   AND owner_membership.role IN ('owner', 'resident')
LEFT JOIN users owner_user ON owner_user.id = owner_membership.user_id
LEFT JOIN whatsapp_threads thread ON thread.unit_id = u.id
WHERE la.id = $1
  AND lp.scheme_id = $2;