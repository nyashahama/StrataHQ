"use client";

import RetryState from "@/components/RetryState";

export default function Error({
  reset,
}: {
  error: Error;
  reset: () => void;
}) {
  return (
    <RetryState
      title="Could not load portfolio overview"
      message="Temporary service issue. Try again."
      onRetry={() => reset()}
    />
  );
}
