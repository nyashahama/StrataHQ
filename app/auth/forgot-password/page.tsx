"use client";
import { useState } from "react";
import Link from "next/link";
import LogoIcon from "@/components/LogoIcon";
import { forgotPasswordAction } from "@/lib/auth-actions";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    await forgotPasswordAction(email);
    setLoading(false);
    setSubmitted(true);
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <div className="flex items-center gap-2 mb-8">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </div>

        {submitted ? (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-3">
              Check your email
            </h1>
            <p className="text-muted text-sm mb-6">
              If an account exists for <strong>{email}</strong>, a password
              reset link has been sent.
            </p>
            <Link
              href="/auth/login"
              className="text-[13px] text-accent font-medium hover:underline"
            >
              Back to login →
            </Link>
          </div>
        ) : (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
              Reset password
            </h1>
            <p className="text-muted text-sm mb-8">
              Enter your email and we&apos;ll send a reset link.
            </p>
            <form onSubmit={handleSubmit} className="space-y-5">
              <div>
                <label
                  htmlFor="email"
                  className="block text-sm font-medium text-ink mb-1"
                >
                  Email
                </label>
                <input
                  id="email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                  placeholder="you@example.com"
                />
              </div>
              <button
                type="submit"
                disabled={loading}
                className="w-full rounded bg-ink text-page py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
              >
                {loading ? "Sending…" : "Send reset link"}
              </button>
            </form>
            <p className="mt-6 text-center text-sm text-muted">
              <Link
                href="/auth/login"
                className="text-accent hover:underline font-medium"
              >
                Back to login
              </Link>
            </p>
          </div>
        )}
      </div>
    </main>
  );
}
