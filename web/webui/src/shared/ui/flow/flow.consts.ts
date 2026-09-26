/** Auto-layout geometry shared by every flow canvas: table nodes form the left column, sinks the right. */
export const TABLE_COLUMN_X = 0
export const SINK_COLUMN_X = 584
/** Vertical offset of a column caption above the column's first node. */
export const LABEL_OFFSET_Y = 36

/** Default duration (ms) an edge stays animated after `animate()` before reverting. */
export const DEFAULT_ANIMATE_DURATION = 1300

/** Cycle (ms) of xyflow's `dashdraw` keyframes on animated edges (`animation: dashdraw 0.5s`). */
export const EDGE_DASH_PERIOD = 500

/** Duration (ms) of the animated zoom and fit-view controls. */
export const ZOOM_DURATION = 200
/** Duration (ms) of the pan that brings a selected node or edge into view. */
export const REVEAL_DURATION = 300
/** Gap (px) kept between a revealed node or edge and the edge of the unobstructed canvas area. */
export const REVEAL_MARGIN = 32
