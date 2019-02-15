import React, {SFC} from 'react'

import {PlotEnv} from 'src/minard'
import {bin} from 'src/minard/utils/bin'
import HistogramBars from 'src/minard/components/HistogramBars'
import HistogramTooltip from 'src/minard/components/HistogramTooltip'
import {findHoveredRowIndices} from 'src/minard/utils/findHoveredRowIndices'
import {useLayer} from 'src/minard/utils/useLayer'
import {useDomain} from 'src/minard/utils/useDomain'

export enum Position {
  Stacked = 'stacked',
  Overlaid = 'overlaid',
}

// first render, no controlled domains
// render plot with skeleton data
// renders child
// child registers stat table with auto computed domain
// parent/env reacts and sets xDomain and yDomain, layout
// child rerenders but was unnecessary :(

export interface Props {
  env: PlotEnv
  x: string
  fill: string[]
  colors: string[]
  position?: Position
  binCount?: number
  tooltip?: (props: TooltipProps) => JSX.Element
}

export interface TooltipProps {
  xMin: number
  xMax: number
  counts: Array<{
    grouping: {[colName: string]: any}
    count: number
    color: string
  }>
}

export const Histogram: SFC<Props> = ({
  env,
  x,
  fill,
  colors,
  tooltip = null,
  binCount = null,
  position = Position.Stacked,
}: Props) => {
  const defaultTable = env.defaults.table
  const xDomain = useDomain(env.xDomain, defaultTable.columns[x])

  const layer = useLayer(
    env,
    () => {
      const [table, aesthetics] = bin(
        defaultTable,
        x,
        xDomain,
        fill,
        binCount,
        position
      )

      return {table, aesthetics, colors, scales: {}}
    },
    [defaultTable, x, fill, position, binCount, colors]
  )

  if (!layer) {
    return null
  }

  const {
    innerWidth,
    innerHeight,
    hoverX,
    hoverY,
    defaults: {
      scales: {x: xScale, y: yScale},
    },
  } = env

  const {aesthetics, table} = layer

  const hoveredRowIndices = findHoveredRowIndices(
    table.columns[aesthetics.xMin],
    table.columns[aesthetics.xMax],
    table.columns[aesthetics.yMax],
    hoverX,
    hoverY,
    xScale,
    yScale
  )

  return (
    <>
      <HistogramBars
        width={innerWidth}
        height={innerHeight}
        layer={layer}
        xScale={xScale}
        yScale={yScale}
        position={position}
        hoveredRowIndices={hoveredRowIndices}
      />
      {hoveredRowIndices && (
        <HistogramTooltip
          hoverX={hoverX}
          hoverY={hoverY}
          hoveredRowIndices={hoveredRowIndices}
          layer={layer}
          tooltip={tooltip}
        />
      )}
    </>
  )
}
