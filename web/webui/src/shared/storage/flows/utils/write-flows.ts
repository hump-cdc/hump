import { db } from '@/shared/storage'

import { FLOW_STORES } from '../schema'
import { NodeResolver } from './node-resolver'

/**
 * Run `body` against every flow store in a single read-write transaction. The callback must
 * `await` `body` rather than return its promise, or Dexie commits before the first write lands
 * (`PrematureCommitError`) — the transaction is bound to the zone the callback runs in.
 */
export const writeFlows = async <T>(body: (resolve: NodeResolver) => Promise<T>): Promise<T> =>
    await db.transaction('rw', FLOW_STORES(), async () => await body(new NodeResolver()))
