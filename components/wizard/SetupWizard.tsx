"use client";
import { useState } from "react";
import LogoIcon from "@/components/LogoIcon";
import { setupAction } from "@/lib/auth-actions";
import { setSessionCookie } from "@/lib/auth";

type Step = 1 | 2 | 3;

export default function SetupWizard() {
  const [step, setStep] = useState<Step>(1);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const [orgName, setOrgName] = useState("");
  const [contactEmail, setContactEmail] = useState("");
  const [schemeName, setSchemeName] = useState("");
  const [address, setAddress] = useState("");
  const [unitCount, setUnitCount] = useState("");

  function handleStep1(e: React.FormEvent) {
    e.preventDefault();
    setStep(2);
  }

  async function handleStep2(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    const result = await setupAction({
      org_name: orgName,
      contact_email: contactEmail,
      scheme_name: schemeName,
      scheme_address: address,
      unit_count: parseInt(unitCount, 10),
    });
    setLoading(false);

    if ("error" in result) {
      setError(result.error);
      return;
    }

    setSessionCookie(result.user);
    setStep(3);
  }

  function handleFinish() {
    window.location.replace("/agent");
  }

  const STEP_LABELS = ["Organisation", "First scheme", "Done"];

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-md py-8 sm:py-12">
        {/* Logo */}
        <div className="flex items-center gap-2 mb-10">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </div>

        {/* Step indicator */}
        <div className="flex flex-wrap items-center gap-2 mb-8">
          {STEP_LABELS.map((label, i) => {
            const s = (i + 1) as Step;
            const active = s === step;
            const done = s < step;
            return (
              <div key={label} className="flex items-center gap-2">
                <div
                  className={`w-6 h-6 rounded-full flex items-center justify-center text-[11px] font-semibold flex-shrink-0 ${done ? "bg-green text-white" : active ? "bg-accent text-white" : "bg-border text-muted"}`}
                >
                  {done ? "✓" : s}
                </div>
                <span
                  className={`text-[12px] hidden sm:inline ${active ? "text-ink font-semibold" : "text-muted"}`}
                >
                  {label}
                </span>
                {i < STEP_LABELS.length - 1 && (
                  <div className="w-8 h-px bg-border mx-1" />
                )}
              </div>
            );
          })}
        </div>

        {step === 1 && (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
              Set up your organisation
            </h1>
            <p className="text-muted text-sm mb-8">
              Tell us about your property management company.
            </p>
            <form onSubmit={handleStep1} className="space-y-5">
              <div>
                <label
                  htmlFor="orgName"
                  className="block text-sm font-medium text-ink mb-1"
                >
                  Organisation name
                </label>
                <input
                  id="orgName"
                  type="text"
                  required
                  value={orgName}
                  onChange={(e) => setOrgName(e.target.value)}
                  placeholder="e.g. Acme Property Management"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              <div>
                <label
                  htmlFor="contactEmail"
                  className="block text-sm font-medium text-ink mb-1"
                >
                  Contact email
                </label>
                <input
                  id="contactEmail"
                  type="email"
                  required
                  value={contactEmail}
                  onChange={(e) => setContactEmail(e.target.value)}
                  placeholder="admin@acme.co.za"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              <button
                type="submit"
                className="w-full rounded bg-ink text-page py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity"
              >
                Continue →
              </button>
            </form>
          </div>
        )}

        {step === 2 && (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
              Add your first scheme
            </h1>
            <p className="text-muted text-sm mb-8">
              You can add more schemes later from your dashboard.
            </p>
            <form onSubmit={handleStep2} className="space-y-5">
              <div>
                <label
                  htmlFor="schemeName"
                  className="block text-sm font-medium text-ink mb-1"
                >
                  Scheme name
                </label>
                <input
                  id="schemeName"
                  type="text"
                  required
                  value={schemeName}
                  onChange={(e) => setSchemeName(e.target.value)}
                  placeholder="e.g. Sunridge Heights"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              <div>
                <label
                  htmlFor="address"
                  className="block text-sm font-medium text-ink mb-1"
                >
                  Physical address
                </label>
                <input
                  id="address"
                  type="text"
                  required
                  value={address}
                  onChange={(e) => setAddress(e.target.value)}
                  placeholder="e.g. 14 Ocean Drive, Cape Town"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              <div>
                <label
                  htmlFor="unitCount"
                  className="block text-sm font-medium text-ink mb-1"
                >
                  Number of units
                </label>
                <input
                  id="unitCount"
                  type="number"
                  min="1"
                  required
                  value={unitCount}
                  onChange={(e) => setUnitCount(e.target.value)}
                  placeholder="e.g. 24"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              {error && <p className="text-sm text-red">{error}</p>}
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => setStep(1)}
                  className="px-4 py-2.5 text-sm font-medium text-muted hover:text-ink border border-border rounded transition-colors"
                >
                  ← Back
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="flex-1 rounded bg-ink text-page py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
                >
                  {loading ? "Creating…" : "Continue →"}
                </button>
              </div>
            </form>
          </div>
        )}

        {step === 3 && (
          <div>
            <div className="bg-green-bg border border-green/20 rounded-xl px-6 py-8 text-center mb-6">
              <div className="text-3xl mb-3">✓</div>
              <h1 className="font-serif text-2xl font-semibold text-ink mb-2">
                You&apos;re all set!
              </h1>
              <p className="text-sm text-muted">
                <strong className="text-ink">{orgName}</strong> and your first
                scheme <strong className="text-ink">{schemeName}</strong> have
                been created. You can now invite trustees and residents from the
                Members page.
              </p>
            </div>
            <button
              onClick={handleFinish}
              className="w-full rounded bg-ink text-page py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity"
            >
              Go to dashboard →
            </button>
          </div>
        )}
      </div>
    </main>
  );
}
