/**
 * The hand-driven live stream. Importing this file swaps the app-wide `liveStream` for it (the
 * `mock.module` in `stream.fake.ts`), so every test whose tree reaches `@/shared/live` imports it,
 * by name or bare (`import '@/shared/testing/live.fake'`).
 */
export { fakeLiveStream } from '../live/stream.fake'
