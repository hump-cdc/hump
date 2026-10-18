import { useEffect, useState } from 'react'

import type { Edge, Node } from '@xyflow/react'

import { LAYOUT_DURATION } from './flow.consts'

/** Position and opacity of a node at some moment of a transition. */
type Pose = { x: number; y: number; opacity: number }

/**
 * A layout transition from the rendered state at `start` towards the latest `nodes`/`edges`.
 * `from`/`to` hold the poses (and edge opacities) of everything rendered, leaving items included;
 * leaving items are the last seen objects of nodes and edges no longer in the targets.
 */
export type FlowTransition = {
    nodes: Node[]
    edges: Edge[]
    signature: string
    start: number
    from: ReadonlyMap<string, Pose>
    to: ReadonlyMap<string, Pose>
    edgeFrom: ReadonlyMap<string, number>
    edgeTo: ReadonlyMap<string, number>
    leavingNodes: Node[]
    leavingEdges: Edge[]
}

const NO_NODES: Node[] = []
const NO_EDGES: Edge[] = []

/**
 * Animate layout changes of `nodes`/`edges`: nodes glide to their new positions, new nodes and
 * edges fade in, removed ones fade out in place before leaving. Positions are tweened in data, so
 * edges follow the moving nodes. Only a change of node ids, positions or edge ids starts a
 * transition; new data for the same layout passes straight through. The first render is not animated.
 */
export const useFlowTransition = (
    nodes: Node[] | undefined,
    edges: Edge[] | undefined,
    duration: number = LAYOUT_DURATION,
): { nodes: Node[]; edges: Edge[] } => {
    const targetNodes = nodes ?? NO_NODES
    const targetEdges = edges ?? NO_EDGES

    const [transition, setTransition] = useState(() => createTransition(targetNodes, targetEdges))
    const [, setFrame] = useState(0)

    // Derived during render, so the new layout never paints at its final positions before tweening.
    let current = transition
    if (targetNodes !== transition.nodes || targetEdges !== transition.edges) {
        current = advanceTransition(transition, targetNodes, targetEdges, performance.now(), duration)
        setTransition(current)
    }

    const { start } = current
    useEffect(() => {
        if (!Number.isFinite(start)) {
            return
        }

        let frame = 0
        const tick = () => {
            if (performance.now() - start < duration) {
                setFrame((value) => value + 1)
                frame = requestAnimationFrame(tick)
                return
            }
            setTransition((prev) => (prev.start === start ? createTransition(prev.nodes, prev.edges) : prev))
        }
        frame = requestAnimationFrame(tick)

        return () => cancelAnimationFrame(frame)
    }, [start, duration])

    return renderTransition(current, getProgress(current.start, performance.now(), duration))
}

/** Settled transition for `nodes`/`edges`: everything fully shown in place, nothing leaving. */
export const createTransition = (nodes: Node[], edges: Edge[]): FlowTransition => ({
    nodes,
    edges,
    signature: getLayoutSignature(nodes, edges),
    start: Number.NEGATIVE_INFINITY,
    from: new Map(),
    to: new Map(nodes.map((node) => [node.id, { ...node.position, opacity: 1 }])),
    edgeFrom: new Map(),
    edgeTo: new Map(edges.map((edge) => [edge.id, 1])),
    leavingNodes: [],
    leavingEdges: [],
})

/**
 * Next transition for new targets at `now`. Same layout: only the targets are swapped. Otherwise a
 * new tween starts from the poses rendered at `now`, so an interrupted transition continues smoothly.
 */
export const advanceTransition = (
    prev: FlowTransition,
    nodes: Node[],
    edges: Edge[],
    now: number,
    duration: number,
): FlowTransition => {
    const signature = getLayoutSignature(nodes, edges)
    if (signature === prev.signature) {
        return { ...prev, nodes, edges }
    }

    const t = getProgress(prev.start, now, duration)
    const from = new Map<string, Pose>()
    const to = new Map<string, Pose>()
    const edgeFrom = new Map<string, number>()
    const edgeTo = new Map<string, number>()

    for (const node of nodes) {
        const target = { ...node.position, opacity: 1 }
        from.set(node.id, getPose(prev, node.id, t) ?? { ...target, opacity: 0 })
        to.set(node.id, target)
    }
    const leavingNodes = collectLeaving(
        [...prev.nodes, ...prev.leavingNodes],
        to,
        (id) => getPose(prev, id, t)?.opacity,
    )
    for (const node of leavingNodes) {
        const pose = getPose(prev, node.id, t) as Pose
        from.set(node.id, pose)
        to.set(node.id, { ...pose, opacity: 0 })
    }

    for (const edge of edges) {
        edgeFrom.set(edge.id, getEdgeOpacity(prev, edge.id, t) ?? 0)
        edgeTo.set(edge.id, 1)
    }
    const leavingEdges = collectLeaving([...prev.edges, ...prev.leavingEdges], edgeTo, (id) =>
        getEdgeOpacity(prev, id, t),
    ).filter((edge) => to.has(edge.source) && to.has(edge.target))
    for (const edge of leavingEdges) {
        edgeFrom.set(edge.id, getEdgeOpacity(prev, edge.id, t) as number)
        edgeTo.set(edge.id, 0)
    }

    return { nodes, edges, signature, start: now, from, to, edgeFrom, edgeTo, leavingNodes, leavingEdges }
}

/**
 * Nodes and edges to render at eased progress `t`: targets at their tweened pose, then leaving
 * items fading out, inert. A settled transition returns the targets as-is.
 */
export const renderTransition = (transition: FlowTransition, t: number): { nodes: Node[]; edges: Edge[] } => {
    const { nodes, edges, leavingNodes, leavingEdges } = transition
    if (t >= 1 && leavingNodes.length === 0 && leavingEdges.length === 0) {
        return { nodes, edges }
    }

    return {
        nodes: [
            ...nodes.map((node) => withPose(node, getPose(transition, node.id, t))),
            ...leavingNodes.map((node) => {
                const posed = withPose(node, getPose(transition, node.id, t))
                return {
                    ...posed,
                    style: { ...posed.style, pointerEvents: 'none' as const },
                    selectable: false,
                    focusable: false,
                }
            }),
        ],
        edges: [
            ...edges.map((edge) => withOpacity(edge, getEdgeOpacity(transition, edge.id, t))),
            ...leavingEdges.map((edge) => ({
                ...withOpacity(edge, getEdgeOpacity(transition, edge.id, t)),
                selectable: false,
                focusable: false,
                interactionWidth: 0,
            })),
        ],
    }
}

/** Eased progress (0–1) of a transition started at `start`; 1 once it is over or never started. */
const getProgress = (start: number, now: number, duration: number): number => {
    const linear = duration > 0 ? Math.min(1, Math.max(0, (now - start) / duration)) : 1
    return linear < 0.5 ? 4 * linear ** 3 : 1 - (-2 * linear + 2) ** 3 / 2
}

/** Ids and positions of every node plus every edge id: what a transition animates. */
const getLayoutSignature = (nodes: Node[], edges: Edge[]): string =>
    [...nodes.map(({ id, position }) => `${id}@${position.x},${position.y}`), ...edges.map(({ id }) => id)].join('\n')

/** Pose of node `id` at progress `t`, or `undefined` when the transition does not render it. */
const getPose = (transition: FlowTransition, id: string, t: number): Pose | undefined => {
    const to = transition.to.get(id)
    const from = transition.from.get(id)
    if (!to || !from) {
        return to
    }

    return { x: lerp(from.x, to.x, t), y: lerp(from.y, to.y, t), opacity: lerp(from.opacity, to.opacity, t) }
}

/** Fade opacity of edge `id` at progress `t`, or `undefined` when the transition does not render it. */
const getEdgeOpacity = (transition: FlowTransition, id: string, t: number): number | undefined => {
    const to = transition.edgeTo.get(id)
    const from = transition.edgeFrom.get(id)
    return to === undefined || from === undefined ? to : lerp(from, to, t)
}

/** Items missing from the new `targets` that are still visible, i.e. have to fade out. */
const collectLeaving = <T extends { id: string }>(
    items: T[],
    targets: ReadonlyMap<string, unknown>,
    getOpacity: (id: string) => number | undefined,
): T[] => items.filter((item) => !targets.has(item.id) && (getOpacity(item.id) ?? 0) > 0)

/** `node` placed at `pose`, its fade merged into its own style. */
const withPose = (node: Node, pose: Pose | undefined): Node =>
    pose ? { ...node, position: { x: pose.x, y: pose.y }, style: { ...node.style, opacity: pose.opacity } } : node

/** `edge` faded by `opacity`, multiplied into its own (weight) opacity. */
const withOpacity = (edge: Edge, opacity: number | undefined): Edge =>
    opacity === undefined
        ? edge
        : { ...edge, style: { ...edge.style, opacity: Number(edge.style?.opacity ?? 1) * opacity } }

/** Linear interpolation from `from` to `to` at `t`. */
const lerp = (from: number, to: number, t: number): number => from + (to - from) * t
