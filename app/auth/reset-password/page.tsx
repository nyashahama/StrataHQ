"use client";

import { useState, Suspense } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import LogoIcon from "@/components/LogoIcon";
import { resetPasswordAction } from "@/lib/auth-actions";

function ResetPasswordForm() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (password !== confirm) {
      setError("Passwords do not match");
      return;
    }
    setError("");
    setLoading(true);
    const result = await resetPasswordAction(token, password);
    setLoading(false);

    if ("error" in result) {
      setError(result.error);
      return;
    }

    setSuccess(true);
  }

  if (!token) {
    return (
      <div>
        <h1 className="font-serif text-2xl font-semibold text-ink mb-3">
          Invalid link
        </h1>
        <p className="text-muted text-sm mb-6">
          This reset link is missing a token.
        </p>
        <Link
          href="/auth/forgot-password"
          className="text-[13px] text-accent font-medium hover:underline"
        >
          Request a new reset link →
        </Link>
      </div>
    );
  }

  if (success) {
    return (
      <div>
        <h1 className="font-serif text-2xl font-semibold text-ink mb-3">
          Password updated
        </h1>
        <p className="text-muted text-sm mb-6">
          Your password has been reset. You can now log in.
        </p>
        <Link
          href="/auth/login"
          className="text-[13px] text-accent font-medium hover:underline"
        >
          Go to login →
        </Link>
      </div>
    );
  }

  return (
    <div>
      <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
        Set new password
      </h1>
      <p className="text-muted text-sm mb-8">
        Enter and confirm your new password below.
      </p>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label
            htmlFor="password"
            className="block text-sm font-medium text-ink mb-1"
          >
            New password
          </label>
          <input
            id="password"
            type="password"
            autoComplete="new-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
            placeholder="••••••••"
          />
        </div>
        <div>
          <label
            htmlFor="confirm"
            className="block text-sm font-medium text-ink mb-1"
          >
            Confirm password
          </label>
          <input
            id="confirm"
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
            placeholder="••••••••"
          />
        </div>
        {error && <p className="text-sm text-red">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded bg-accent text-white py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
        >
          {loading ? "Updating…" : "Update password"}
        </button>
      </form>
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <div className="flex items-center gap-2 mb-8">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </div>
        <Suspense fallback={null}>
          <ResetPasswordForm />
        </Suspense>
      </div>
    </main>
  );
}
