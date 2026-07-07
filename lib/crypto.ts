import { createHmac, timingSafeEqual } from "crypto";

const JWT_SECRET = () => process.env.JWT_SECRET ?? "dev-secret-change-me";

export function signPayload(payload: string): string {
  const secret = JWT_SECRET();
  const hmac = createHmac("sha256", secret).update(payload).digest();
  return Buffer.from(hmac).toString("base64url") + "." + payload;
}

export function verifyPayload(signed: string): string | null {
  const secret = JWT_SECRET();
  const dotIdx = signed.indexOf(".");
  if (dotIdx === -1) return null;

  const sigB64 = signed.substring(0, dotIdx);
  const payload = signed.substring(dotIdx + 1);

  const expected = createHmac("sha256", secret).update(payload).digest();
  const sigBuf = Buffer.from(sigB64, "base64url");

  if (sigBuf.length !== expected.length) return null;
  if (!timingSafeEqual(sigBuf, expected)) return null;

  return payload;
}
