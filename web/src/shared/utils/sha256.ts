/** Compute the SHA-256 hex digest of a string via the Web Crypto API. */
export const sha256 = async (input: string): Promise<string> => {
    const bytes = new TextEncoder().encode(input)
    const digest = await crypto.subtle.digest('SHA-256', bytes)
    return Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, '0')).join('')
}
