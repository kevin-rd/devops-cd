import { useMemo, useCallback, useEffect, useRef, useState } from 'react'
import ReactFlow, {
  Node,
  Edge,
  MiniMap,
  MarkerType,
} from 'reactflow'
import dagre from 'dagre'
import { Empty } from 'antd'
import type { ReleaseApp, AppTypeConfigInfo } from '@/types'
import AppNode from './AppNode'
import GroupNode from './GroupNode'
import 'reactflow/dist/style.css'
import styles from './DependencyGraph.module.css'

interface DependencyGraphProps {
  releaseApps: ReleaseApp[]
  appTypeConfigs?: Record<string, AppTypeConfigInfo>
}

const nodeTypes = {
  appNode: AppNode,
  groupNode: GroupNode,
}

// 布局配置参数
const nodeWidth = 180     // 节点宽度
const nodeHeight = 40     // 节点高度（只有名字，更小）
const ranksep = 160       // 层间距
const nodesep = 50        // 节点间距
const TRANSLATE_PADDING = 50  // 移动时，保持节点在屏幕内
const LEFT_PADDING = 0
const MIN_CONTAINER_WIDTH = 1200
const TOP_PADDING = 20
const LAYER_EXTRA_PADDING = 100
const ROW_GAP = 40
const MAX_NODES_PER_ROW = 8 // 增加每行最大节点数
const GROUP_PADDING = 20

// 边-节点交叉检测的辅助函数
interface NodeRect {
  id: string
  x: number
  y: number
  width: number
  height: number
}

interface EdgeLine {
  sourceId: string
  targetId: string
  sourceX: number
  sourceY: number
  targetX: number
  targetY: number
}

// 检查边是否与节点矩形相交（简化的贝塞尔曲线近似为直线）
const checkEdgeNodeIntersection = (edge: EdgeLine, node: NodeRect): boolean => {
  // 如果边的起点或终点是该节点，不算交叉
  if (edge.sourceId === node.id || edge.targetId === node.id) {
    return false
  }

  // 边的中心点（贝塞尔曲线的近似中点）
  const edgeCenterX = (edge.sourceX + edge.targetX) / 2
  const edgeCenterY = (edge.sourceY + edge.targetY) / 2

  // 简化判断：检查边的中点是否在节点的扩展区域内
  const margin = 10 // 给一些容错空间
  const inXRange = edgeCenterX >= node.x - margin && edgeCenterX <= node.x + node.width + margin
  const inYRange = edgeCenterY >= node.y - margin && edgeCenterY <= node.y + node.height + margin

  if (!inXRange || !inYRange) {
    return false
  }

  // 更精确的判断：检查线段是否穿过矩形
  return checkLineRectIntersection(
    edge.sourceX,
    edge.sourceY,
    edge.targetX,
    edge.targetY,
    node.x - margin,
    node.y - margin,
    node.width + margin * 2,
    node.height + margin * 2
  )
}

// 检查线段是否与矩形相交
const checkLineRectIntersection = (
  x1: number,
  y1: number,
  x2: number,
  y2: number,
  rectX: number,
  rectY: number,
  rectW: number,
  rectH: number
): boolean => {
  // 检查线段的两个端点是否在矩形内
  const p1Inside = x1 >= rectX && x1 <= rectX + rectW && y1 >= rectY && y1 <= rectY + rectH
  const p2Inside = x2 >= rectX && x2 <= rectX + rectW && y2 >= rectY && y2 <= rectY + rectH

  if (p1Inside || p2Inside) {
    return true
  }

  // 检查线段是否与矩形的四条边相交
  const rectLeft = rectX
  const rectRight = rectX + rectW
  const rectTop = rectY
  const rectBottom = rectY + rectH

  return (
    checkLineIntersection(x1, y1, x2, y2, rectLeft, rectTop, rectRight, rectTop) ||
    checkLineIntersection(x1, y1, x2, y2, rectRight, rectTop, rectRight, rectBottom) ||
    checkLineIntersection(x1, y1, x2, y2, rectRight, rectBottom, rectLeft, rectBottom) ||
    checkLineIntersection(x1, y1, x2, y2, rectLeft, rectBottom, rectLeft, rectTop)
  )
}

// 检查两条线段是否相交
const checkLineIntersection = (
  x1: number,
  y1: number,
  x2: number,
  y2: number,
  x3: number,
  y3: number,
  x4: number,
  y4: number
): boolean => {
  const denominator = (x1 - x2) * (y3 - y4) - (y1 - y2) * (x3 - x4)
  if (Math.abs(denominator) < 0.0001) {
    return false // 平行或重合
  }

  const t = ((x1 - x3) * (y3 - y4) - (y1 - y3) * (x3 - x4)) / denominator
  const u = -((x1 - x2) * (y1 - y3) - (y1 - y2) * (x1 - x3)) / denominator

  return t >= 0 && t <= 1 && u >= 0 && u <= 1
}

interface LayerLayout {
  level: number
  label: string
  types: string[]
  left: number
  right: number
  top: number
  bottom: number
}

// 创建 dagre 图实例（全局复用，保持布局稳定）
const dagreGraph = new dagre.graphlib.Graph()
dagreGraph.setDefaultEdgeLabel(() => ({}))

/**
 * 使用 Dagre 进行图布局
 * - TB (Top-Bottom): 从上到下布局，符合"依赖方向从上到下"的需求
 * - 自动分层减少交叉
 * - 保持布局稳定（节点ID作为唯一标识）
 */
const getLayoutedElements = (nodes: Node[], edges: Edge[], appTypeLevels: Map<string, number>) => {
  if (nodes.length === 0) return { nodes: [], edges }

  dagreGraph.setGraph({
    rankdir: 'TB',
    ranksep: ranksep,
    nodesep: nodesep,
    edgesep: 50,
    ranker: 'network-simplex', // 最优化边长度和交叉
    align: 'UL', // 左上对齐
    acyclicer: 'greedy',
    marginx: 20,
    marginy: 20,
  })

  dagreGraph.nodes().forEach((n) => dagreGraph.removeNode(n))
  dagreGraph.edges().forEach((e) => dagreGraph.removeEdge(e.v, e.w))

  nodes.forEach((node) => {
    const appType = node.data?.releaseApp?.app_type as string | undefined
    const rank = appType ? appTypeLevels.get(appType) ?? 0 : 0

    dagreGraph.setNode(node.id, {
      width: nodeWidth,
      height: nodeHeight,
      rank,
      label: node.id,
    })
  })

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target, {
      weight: 1,
      minlen: 1,
    })
  })

  dagre.layout(dagreGraph)

  const layoutedNodes = nodes.map((node) => {
    const dagreNode = dagreGraph.node(node.id)

    return {
      ...node,
      position: {
        x: dagreNode.x - nodeWidth / 2,
        y: dagreNode.y - nodeHeight / 2,
      },
    }
  })

  const minX = Math.min(...layoutedNodes.map((n) => n.position.x))
  const minY = Math.min(...layoutedNodes.map((n) => n.position.y))

  const normalizedNodes = layoutedNodes.map((node) => ({
    ...node,
    position: {
      x: node.position.x - minX + 20,
      y: node.position.y - minY + 20,
    },
  }))

  return { nodes: normalizedNodes, edges }
}

// 计算图的实际高度和宽度
const calculateGraphDimensions = (nodes: Node[]): { width: number; height: number } => {
  if (nodes.length === 0) return { width: 800, height: 400 }

  let maxX = 0
  let maxY = 0

  nodes.forEach((node) => {
    const width = (typeof node.style?.width === 'number' ? node.style?.width : node.data?.width) ?? nodeWidth
    const height = (typeof node.style?.height === 'number' ? node.style?.height : node.data?.height) ?? nodeHeight
    maxX = Math.max(maxX, node.position.x + width)
    maxY = Math.max(maxY, node.position.y + height)
  })

  return {
    width: maxX + 20,
    height: maxY + 20,
  }
}

const adjustNodesToWidth = (
  nodes: Node[],
  edges: Edge[],
  containerWidth: number,
  appTypeLevels: Map<string, number>,
  appTypeConfigs?: Record<string, AppTypeConfigInfo>,
): { nodes: Node[]; layers: LayerLayout[]; totalHeight: number } => {
  if (nodes.length === 0) {
    return { nodes, layers: [], totalHeight: TOP_PADDING + nodeHeight }
  }

  const safeWidth = Math.max(containerWidth, MIN_CONTAINER_WIDTH)
  const innerPadding = LEFT_PADDING === 0 ? 20 : LEFT_PADDING // 固定左边距，不再基于容器宽度百分比
  const availableWidth = safeWidth - innerPadding * 2
  const maxNodesBasedOnWidth = Math.max(1, Math.ceil((availableWidth + nodesep) / (nodeWidth + nodesep)))
  const effectiveMaxPerRow = Math.max(1, Math.min(MAX_NODES_PER_ROW, maxNodesBasedOnWidth))

  const minX = Math.min(...nodes.map((node) => node.position.x))
  const maxX = Math.max(...nodes.map((node) => node.position.x))
  const currentWidth = maxX - minX + nodeWidth

  // 只在节点过多时进行缩放，不再进行扩展
  let scale = 1
  if (currentWidth > safeWidth * 0.95) {
    scale = (safeWidth * 0.95) / currentWidth
  }

  const scaledPositions = new Map<string, { x: number; level: number; appType?: string }>()

  nodes.forEach((node) => {
    // 直接从左边距开始，不进行居中对齐
    const scaledX = (node.position.x - minX) * scale + innerPadding
    const appType = node.data?.releaseApp?.app_type
    const level = appType ? appTypeLevels.get(appType) ?? 0 : 0
    scaledPositions.set(node.id, {
      x: scaledX,
      level,
      appType,
    })
  })

  // 按 app_type level 分组
  const layerGroups = new Map<number, string[]>()
  scaledPositions.forEach((pos, id) => {
    const group = layerGroups.get(pos.level) || []
    group.push(id)
    layerGroups.set(pos.level, group)
  })

  const finalPositions = new Map<string, { x: number; y: number }>()
  const layerLayouts: LayerLayout[] = []
  const sortedLayerKeys = Array.from(layerGroups.keys()).sort((a, b) => a - b)

  let lastLayerBottom = TOP_PADDING

  sortedLayerKeys.forEach((layerKey) => {
    const layerNodeIds = layerGroups.get(layerKey) || []
    if (layerNodeIds.length === 0) {
      return
    }

    // 构建该大类型内部的依赖图
    const adjacency = new Map<string, Set<string>>()
    const inDegree = new Map<string, number>()

    layerNodeIds.forEach((id) => {
      adjacency.set(id, new Set())
      inDegree.set(id, 0)
    })

    edges.forEach((edge) => {
      const source = scaledPositions.get(edge.source)
      const target = scaledPositions.get(edge.target)
      if (!source || !target) {
        return
      }
      // 只处理大类型内部的依赖关系
      if (source.level !== layerKey || target.level !== layerKey) {
        return
      }
      if (!adjacency.has(edge.source) || !adjacency.has(edge.target)) {
        return
      }
      if (!adjacency.get(edge.source)!.has(edge.target)) {
        adjacency.get(edge.source)!.add(edge.target)
        inDegree.set(edge.target, (inDegree.get(edge.target) ?? 0) + 1)
      }
    })

    // 使用 BFS 分层（层内拓扑排序）
    const subLayers: string[][] = []
    const visited = new Set<string>()
    const nodeToSubLayer = new Map<string, number>()

    let currentSubLayer = 0
    while (visited.size < layerNodeIds.length) {
      const currentLevelNodes: string[] = []

      layerNodeIds.forEach((id) => {
        if (visited.has(id)) return
        const deg = inDegree.get(id) ?? 0
        if (deg === 0) {
          currentLevelNodes.push(id)
        }
      })

      if (currentLevelNodes.length === 0) {
        // 有环或剩余节点，直接加入
        layerNodeIds.forEach((id) => {
          if (!visited.has(id)) {
            currentLevelNodes.push(id)
          }
        })
      }

      currentLevelNodes.forEach((id) => {
        visited.add(id)
        nodeToSubLayer.set(id, currentSubLayer)
        adjacency.get(id)?.forEach((neighbor) => {
          const deg = (inDegree.get(neighbor) ?? 0) - 1
          inDegree.set(neighbor, deg)
        })
      })

      subLayers.push(currentLevelNodes)
      currentSubLayer++
    }

    // 现在 subLayers[i] 是大类型内的第 i 个子层
    let layerLeft = Number.POSITIVE_INFINITY
    let layerRight = Number.NEGATIVE_INFINITY
    let currentY = lastLayerBottom
    const layerStartY = lastLayerBottom

    subLayers.forEach((subLayerNodeIds, subLayerIndex) => {
      if (subLayerNodeIds.length === 0) return

      // 获取这些节点的信息
      const subLayerNodes = subLayerNodeIds.map((id) => {
        const pos = scaledPositions.get(id)!
        // 计算依赖边
        const incomingEdges = edges.filter((e) => e.target === id && layerNodeIds.includes(e.source))
        const outgoingEdges = edges.filter((e) => e.source === id && layerNodeIds.includes(e.target))

        // 计算跨层边（与其他大类型的依赖）
        const crossLayerIncoming = edges.filter((e) => e.target === id && !layerNodeIds.includes(e.source))
        const crossLayerOutgoing = edges.filter((e) => e.source === id && !layerNodeIds.includes(e.target))

        // 总依赖数（用于排序）
        const totalIncoming = incomingEdges.length + crossLayerIncoming.length
        const totalOutgoing = outgoingEdges.length + crossLayerOutgoing.length

        return {
          id,
          dagreX: pos.x, // 保留 Dagre 的 X 坐标作为参考
          appType: pos.appType,
          outgoing: totalOutgoing,
          incoming: totalIncoming,
          incomingEdges,
          outgoingEdges,
        }
      })

      // 计算每个节点应该的 preferredX（基于其依赖节点的位置）
      const nodePreferredX = new Map<string, number>()
      
      subLayerNodes.forEach((nodeData) => {
        let preferredX = nodeData.dagreX
        
        // 如果有来自上一个子层的依赖，尽量靠近它们
        if (subLayerIndex > 0 && nodeData.incomingEdges.length > 0) {
          const parentXs = nodeData.incomingEdges
            .map((e) => {
              const sourcePos = finalPositions.get(e.source)
              return sourcePos ? sourcePos.x : null
            })
            .filter((x): x is number => x !== null)
          
          if (parentXs.length > 0) {
            preferredX = parentXs.reduce((sum, x) => sum + x, 0) / parentXs.length
          }
        }
        
        nodePreferredX.set(nodeData.id, preferredX)
      })

      // 排序逻辑：
      // 1. 有依赖的（outgoing > 0）排前面（往左）
      // 2. 出度大的排前面（更多依赖的往左）
      // 3. 同 app_type 聚在一起
      // 4. 按 preferredX 排序（尽量靠近依赖的节点）
      subLayerNodes.sort((a, b) => {
        const aHasDeps = a.outgoing > 0
        const bHasDeps = b.outgoing > 0
        
        if (aHasDeps !== bHasDeps) {
          return bHasDeps ? 1 : -1 // 有依赖的排前面
        }
        
        if (a.outgoing !== b.outgoing) {
          return b.outgoing - a.outgoing // 出度大的排前面
        }
        
        if (a.appType !== b.appType) {
          return (a.appType || '').localeCompare(b.appType || '')
        }
        
        return (nodePreferredX.get(a.id) ?? a.dagreX) - (nodePreferredX.get(b.id) ?? b.dagreX)
      })

      // 分多行排列的新策略：
      // 1. 入度大的放在前面的行（上面），出度大的放在后面的行（下面）
      // 2. 优化边与节点的交叉
      const numRows = Math.ceil(subLayerNodes.length / effectiveMaxPerRow)
      const rows: typeof subLayerNodes[] = Array.from({ length: numRows }, () => [])

      // 按综合得分分配到不同行
      // 入度大的优先放前面的行，出度大的优先放后面的行
      subLayerNodes.forEach((nodeData) => {
        // 计算节点应该在的行：入度大 -> 前面，出度大 -> 后面
        // score 越小越靠前
        const score = nodeData.outgoing - nodeData.incoming
        
        // 归一化到 [0, numRows-1]
        const allScores = subLayerNodes.map((n) => n.outgoing - n.incoming)
        const minScore = Math.min(...allScores)
        const maxScore = Math.max(...allScores)
        
        let targetRow = 0
        if (maxScore > minScore) {
          const normalizedScore = (score - minScore) / (maxScore - minScore)
          targetRow = Math.floor(normalizedScore * (numRows - 1))
        }
        
        // 找到目标行或最近的未满行
        let assignedRow = targetRow
        while (assignedRow < numRows && rows[assignedRow].length >= effectiveMaxPerRow) {
          assignedRow++
        }
        
        // 如果所有后面的行都满了，往前找
        if (assignedRow >= numRows) {
          assignedRow = targetRow
          while (assignedRow >= 0 && rows[assignedRow].length >= effectiveMaxPerRow) {
            assignedRow--
          }
        }
        
        // 如果还是没找到，放到第一个有空间的行
        if (assignedRow < 0 || assignedRow >= numRows) {
          assignedRow = rows.findIndex((row) => row.length < effectiveMaxPerRow)
          if (assignedRow === -1) {
            assignedRow = 0 // 降级到第一行
          }
        }
        
        rows[assignedRow].push(nodeData)
      })

      // 为每行的节点计算位置，并考虑边-节点交叉优化
      const tempPositions = new Map<string, { x: number; y: number }>()
      
      rows.forEach((rowNodes, rowIndex) => {
        if (rowNodes.length === 0) return

        // 每行从左到右依次放置
        let currentX = innerPadding
        const y = currentY + rowIndex * (nodeHeight + ROW_GAP)

        rowNodes.forEach((nodeData) => {
          tempPositions.set(nodeData.id, { x: currentX, y })
          currentX += nodeWidth + nodesep
        })
      })

      // 检查并优化边-节点交叉（简单的微调策略）
      // 构建当前子层及之前所有节点的边列表
      const edgeLines: EdgeLine[] = []
      edges.forEach((edge) => {
        const sourcePos = finalPositions.get(edge.source) || tempPositions.get(edge.source)
        const targetPos = finalPositions.get(edge.target) || tempPositions.get(edge.target)
        
        if (sourcePos && targetPos) {
          edgeLines.push({
            sourceId: edge.source,
            targetId: edge.target,
            sourceX: sourcePos.x + nodeWidth / 2,
            sourceY: sourcePos.y + nodeHeight,
            targetX: targetPos.x + nodeWidth / 2,
            targetY: targetPos.y,
          })
        }
      })

      // 统计每个位置的交叉数，尝试局部调整
      rows.forEach((rowNodes) => {
        if (rowNodes.length === 0) return

        rowNodes.forEach((nodeData) => {
          const pos = tempPositions.get(nodeData.id)
          if (!pos) return

          const nodeRect: NodeRect = {
            id: nodeData.id,
            x: pos.x,
            y: pos.y,
            width: nodeWidth,
            height: nodeHeight,
          }

          // 计算当前位置的交叉数
          let intersectionCount = 0
          edgeLines.forEach((edgeLine) => {
            if (checkEdgeNodeIntersection(edgeLine, nodeRect)) {
              intersectionCount++
            }
          })

          // 如果交叉较多，尝试微调 X 位置（在不超出行范围的情况下）
          if (intersectionCount > 2) {
            const adjustments = [-nodesep / 2, nodesep / 2, -nodesep / 4, nodesep / 4]
            let bestX = pos.x
            let minIntersections = intersectionCount

            adjustments.forEach((adjustment) => {
              const testX = pos.x + adjustment
              if (testX < innerPadding || testX + nodeWidth > safeWidth - innerPadding) {
                return
              }

              const testRect: NodeRect = {
                ...nodeRect,
                x: testX,
              }

              let testIntersections = 0
              edgeLines.forEach((edgeLine) => {
                if (checkEdgeNodeIntersection(edgeLine, testRect)) {
                  testIntersections++
                }
              })

              if (testIntersections < minIntersections) {
                minIntersections = testIntersections
                bestX = testX
              }
            })

            tempPositions.set(nodeData.id, { x: bestX, y: pos.y })
          }
        })
      })

      // 应用最终位置
      tempPositions.forEach((pos, id) => {
        finalPositions.set(id, pos)
        layerLeft = Math.min(layerLeft, pos.x)
        layerRight = Math.max(layerRight, pos.x + nodeWidth)
      })

      // 子层间距
      currentY += numRows * nodeHeight + Math.max(0, numRows - 1) * ROW_GAP
      if (subLayerIndex < subLayers.length - 1) {
        currentY += ranksep / 2 // 子层间距
      }
    })

    if (!Number.isFinite(layerLeft) || !Number.isFinite(layerRight)) {
      layerLeft = innerPadding
      layerRight = innerPadding + nodeWidth
    }

    lastLayerBottom = currentY + LAYER_EXTRA_PADDING

    const typeSet = new Set<string>()
    layerNodeIds.forEach((nodeId) => {
      const pos = scaledPositions.get(nodeId)
      if (pos?.appType) {
        typeSet.add(pos.appType)
      }
    })

    const types = Array.from(typeSet).sort()
    const layerLabels = types.map((type) => appTypeConfigs?.[type]?.label || type)
    const label = layerLabels.length > 0 ? layerLabels.join(' / ') : `Layer ${layerKey}`

    layerLayouts.push({
      level: layerKey,
      types,
      label,
      left: layerLeft - GROUP_PADDING,
      right: layerRight + GROUP_PADDING,
      top: layerStartY - GROUP_PADDING,
      bottom: currentY + GROUP_PADDING,
    })
  })

  const positionedNodes = nodes.map((node) => {
    const finalPosition = finalPositions.get(node.id)
    if (!finalPosition) {
      const fallback = scaledPositions.get(node.id)
      if (fallback) {
        const baseY = TOP_PADDING + (fallback.level ?? 0) * (nodeHeight + ranksep)
        return {
          ...node,
          position: {
            x: fallback.x,
            y: baseY,
          },
        }
      }
      return node
    }

    return {
      ...node,
      position: finalPosition,
    }
  })

  const totalHeight = layerLayouts.length
    ? Math.max(...layerLayouts.map((layer) => layer.bottom)) + LAYER_EXTRA_PADDING
    : TOP_PADDING + nodeHeight

  return { nodes: positionedNodes, layers: layerLayouts, totalHeight }
}

// 构建图数据
const buildGraphData = (releaseApps: ReleaseApp[]): { nodes: Node[]; edges: Edge[] } => {
  const nodes: Node[] = []
  const edges: Edge[] = []
  const appIdToIndex = new Map<number, number>()

  // 统计每个应用的入度和出度，判断是否为游离节点
  const inDegree = new Map<number, number>()
  const outDegree = new Map<number, number>()

  releaseApps.forEach((app) => {
    appIdToIndex.set(app.app_id, nodes.length)
    inDegree.set(app.app_id, 0)
    outDegree.set(app.app_id, 0)
  })

  // 计算度数
  releaseApps.forEach((app) => {
    const defaultDepends = app.default_depends_on || []
    const tempDepends = app.temp_depends_on || []
    const allDepends = [...new Set([...defaultDepends, ...tempDepends])]

    allDepends.forEach((depId) => {
      if (appIdToIndex.has(depId)) {
        outDegree.set(app.app_id, (outDegree.get(app.app_id) || 0) + 1)
        inDegree.set(depId, (inDegree.get(depId) || 0) + 1)
      }
    })
  })

  // 创建节点
  releaseApps.forEach((app) => {
    const isIsolated = (inDegree.get(app.app_id) || 0) === 0 && (outDegree.get(app.app_id) || 0) === 0

    nodes.push({
      id: String(app.app_id),
      type: 'appNode',
      position: { x: 0, y: 0 }, // 初始位置，后续由 dagre 计算
      data: {
        releaseApp: app,
        isIsolated,
      },
    })
  })

  // 创建边
  releaseApps.forEach((app) => {
    const defaultDepends = app.default_depends_on || []
    const tempDepends = app.temp_depends_on || []

    // 默认依赖 - 紫色实线
    defaultDepends.forEach((depId) => {
      if (appIdToIndex.has(depId)) {
        edges.push({
          id: `default-${depId}-${app.app_id}`,
          source: String(depId),
          target: String(app.app_id),
          type: 'smoothstep', // 使用 smoothstep 让箭头自动跟随方向
          animated: false,
          style: {
            stroke: '#722ed1',
            strokeWidth: 2,
          },
          markerEnd: {
            type: MarkerType.Arrow,
            color: '#722ed1',
            width: 16,
            height: 10,
            strokeWidth: 2,
          },
          zIndex: -1,
        })
      }
    })

    // 临时依赖 - 蓝色虚线
    tempDepends.forEach((depId) => {
      if (appIdToIndex.has(depId) && !defaultDepends.includes(depId)) {
        edges.push({
          id: `temp-${depId}-${app.app_id}`,
          source: String(depId),
          target: String(app.app_id),
          type: 'default', // 使用 smoothstep 让箭头自动跟随方向
          animated: false,
          style: {
            stroke: '#1890ff',
            strokeWidth: 2,
            strokeDasharray: '6,4',
          },
          markerEnd: {
            type: MarkerType.Arrow,
            color: '#1890ff',
            width: 16,
            height: 10,
            strokeWidth: 2,
          },
          zIndex: -1,
        })
      }
    })
  })

  return { nodes, edges }
}

const buildLayerGroupNodes = (layers: LayerLayout[]): Node[] => {
  return layers.map((layer) => {
    const width = Math.max(layer.right - layer.left, nodeWidth)
    const height = Math.max(layer.bottom - layer.top, nodeHeight)

    return {
      id: `layer-${layer.level}`,
      type: 'groupNode',
      position: { x: layer.left, y: layer.top },
      data: {
        label: layer.label,
        width,
        height,
      },
      style: {
        width,
        height,
        pointerEvents: 'none',
        zIndex: 0,
      },
      draggable: false,
      selectable: false,
      focusable: false,
      connectable: false,
    }
  })
}

const computeAppTypeLevels = (
  releaseApps: ReleaseApp[],
  appTypeConfigs?: Record<string, AppTypeConfigInfo>,
): Map<string, number> => {
  const levels = new Map<string, number>()

  const ensureType = (type: string, adjacency: Map<string, Set<string>>, inDegree: Map<string, number>) => {
    if (!adjacency.has(type)) {
      adjacency.set(type, new Set())
    }
    if (!inDegree.has(type)) {
      inDegree.set(type, 0)
    }
  }

  const types = new Set<string>()
  releaseApps.forEach((app) => {
    if (app.app_type) {
      types.add(app.app_type)
    }
  })

  if (appTypeConfigs) {
    Object.entries(appTypeConfigs).forEach(([type, cfg]) => {
      types.add(type)
      ;(cfg?.dependencies ?? []).forEach((dep) => types.add(dep))
    })
  }

  const adjacency = new Map<string, Set<string>>()
  const inDegree = new Map<string, number>()

  types.forEach((type) => ensureType(type, adjacency, inDegree))

  if (appTypeConfigs) {
    Object.entries(appTypeConfigs).forEach(([type, cfg]) => {
      ensureType(type, adjacency, inDegree)
      const dependencies = cfg?.dependencies ?? []
      dependencies.forEach((dep) => {
        ensureType(dep, adjacency, inDegree)
        if (!adjacency.get(dep)!.has(type)) {
          adjacency.get(dep)!.add(type)
          inDegree.set(type, (inDegree.get(type) ?? 0) + 1)
        }
      })
    })
  }

  const queue: string[] = []
  inDegree.forEach((deg, type) => {
    if (deg === 0) {
      levels.set(type, 0)
      queue.push(type)
    }
  })

  if (queue.length === 0) {
    types.forEach((type) => {
      if (!levels.has(type)) {
        levels.set(type, 0)
        queue.push(type)
      }
    })
  }

  while (queue.length > 0) {
    const current = queue.shift()!
    const currentLevel = levels.get(current) ?? 0

    adjacency.get(current)?.forEach((neighbor) => {
      const candidateLevel = currentLevel + 1
      const prevLevel = levels.get(neighbor) ?? 0
      if (candidateLevel > prevLevel) {
        levels.set(neighbor, candidateLevel)
      }

      const updatedDegree = (inDegree.get(neighbor) ?? 0) - 1
      inDegree.set(neighbor, updatedDegree)
      if (updatedDegree <= 0) {
        queue.push(neighbor)
      }
    })
  }

  types.forEach((type) => {
    if (!levels.has(type)) {
      levels.set(type, 0)
    }
  })

  return levels
}

export default function DependencyGraph({ releaseApps, appTypeConfigs }: DependencyGraphProps) {
  const wrapperRef = useRef<HTMLDivElement | null>(null)
  const [containerWidth, setContainerWidth] = useState<number>(() => (typeof window !== 'undefined' ? window.innerWidth : 1200))

  useEffect(() => {
    const element = wrapperRef.current
    if (!element || typeof ResizeObserver === 'undefined') {
      return
    }

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (entry) {
        setContainerWidth(entry.contentRect.width)
      }
    })

    observer.observe(element)

    return () => observer.disconnect()
  }, [])

  const appTypeLevels = useMemo(() => computeAppTypeLevels(releaseApps, appTypeConfigs), [releaseApps, appTypeConfigs])

  const { nodes, edges, layoutHeight } = useMemo(() => {
    if (releaseApps.length === 0) {
      return { nodes: [], edges: [], layoutHeight: TOP_PADDING + nodeHeight }
    }

    const { nodes: baseNodes, edges: baseEdges } = buildGraphData(releaseApps)
    const { nodes: layoutedNodes, edges } = getLayoutedElements(baseNodes, baseEdges, appTypeLevels)
    const {
      nodes: positionedNodes,
      layers,
      totalHeight,
    } = adjustNodesToWidth(layoutedNodes, edges, containerWidth, appTypeLevels, appTypeConfigs)
    const groupNodes = buildLayerGroupNodes(layers)

    return {
      nodes: [...groupNodes, ...positionedNodes],
      edges,
      layoutHeight: totalHeight,
    }
  }, [releaseApps, appTypeConfigs, containerWidth, appTypeLevels])

  const graphDimensions = useMemo(() => {
    return calculateGraphDimensions(nodes)
  }, [nodes])

  const containerHeight = useMemo(
    () => Math.max(graphDimensions.height, layoutHeight),
    [graphDimensions.height, layoutHeight],
  )
  const translateExtent = useMemo<[[number, number], [number, number]]>(
    () => [
      [-TRANSLATE_PADDING, -TRANSLATE_PADDING],
      [graphDimensions.width + TRANSLATE_PADDING, graphDimensions.height + TRANSLATE_PADDING],
    ],
    [graphDimensions.height, graphDimensions.width],
  )

  const onNodeClick = useCallback((_event: React.MouseEvent, node: Node) => {
    console.log('Node clicked:', node.data.releaseApp)
    // TODO: 打开详情面板
  }, [])

  if (releaseApps.length === 0) {
    return (
      <div className={styles.emptyContainer}>
        <Empty description="暂无应用数据" />
      </div>
    )
  }

  return (
    <div ref={wrapperRef} className={styles.graphWrapper}>
      <div
        className={styles.graphContainer}
        style={{
          height: `${containerHeight}px`,
        }}
      >
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodeClick={onNodeClick}
          nodeTypes={nodeTypes}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={false}
          panOnDrag={true}
          panOnScroll={false}
          zoomOnScroll={false}
          zoomOnPinch={false}
          zoomOnDoubleClick={false}
          preventScrolling={false}
          defaultViewport={{ x: 0, y: 0, zoom: 1 }}
          translateExtent={translateExtent}
          defaultEdgeOptions={{
            type: 'default',
          }}
        >
          <MiniMap
            nodeColor={(node) => {
              if (node.type === 'groupNode') return 'rgba(0,0,0,0)'
              if (node.data?.isIsolated) return '#faad14'
              return '#1890ff'
            }}
            maskColor="rgba(0, 0, 0, 0.05)"
            style={{
              backgroundColor: 'rgba(245, 245, 245, 0.7)',
              backdropFilter: 'blur(4px)',
            }}
          />
        </ReactFlow>
      </div>
    </div>
  )
}

