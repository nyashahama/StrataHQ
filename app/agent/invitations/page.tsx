export default function InvitationsPage() {
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Invitations</h1>
      <p className="text-[14px] text-muted mb-8">Pending trustee and resident invitations.</p>
      <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
        No pending invitations
      </div>
    </div>
  )
}
