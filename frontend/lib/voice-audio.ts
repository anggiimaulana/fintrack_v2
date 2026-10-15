/**
 * Audio encoding helpers for the server transcription path — pure and testable.
 */

/**
 * The backend rejects audio larger than 5 MB (after base64 decoding). We compare
 * the ENCODED length (base64 is ~4/3 of raw), so the ceiling here is slightly
 * below the raw limit; the exact bound matters less than rejecting early with a
 * friendly message instead of a 400.
 */
export const MAX_AUDIO_BASE64 = 5 * 1024 * 1024;

/**
 * blobToBase64 converts a recorded Blob into a bare base64 string (no
 * `data:<mime>;base64,` prefix), which is the shape the API expects.
 */
export function blobToBase64(blob: Blob): Promise<string> {
  const { promise, resolve, reject } = Promise.withResolvers<string>();
  const reader = new FileReader();
  reader.onerror = () => reject(reader.error ?? new Error("failed to read audio blob"));
  reader.onload = () => {
    const result = String(reader.result ?? "");
    const comma = result.indexOf(",");
    resolve(comma >= 0 ? result.slice(comma + 1) : result);
  };
  reader.readAsDataURL(blob);
  return promise;
}

/** isAudioTooLarge reports whether an encoded payload exceeds the server limit. */
export function isAudioTooLarge(base64: string): boolean {
  return base64.length > MAX_AUDIO_BASE64;
}
