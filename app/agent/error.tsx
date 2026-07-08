"use client";

import RetryState from "@/components/RetryState";

export default function Error({
  error,
  reset,
}: {
  error: Error;
  reset: () => void;
}) {
  return (
    <RetryState
      title="Could not load portfolio overview"
      message={error.message || "Temporary service issue. Try again."}
      onRetry={() => reset()}
    />
  );
}
