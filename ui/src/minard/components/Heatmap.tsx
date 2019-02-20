import React, {SFC} from 'react'

import {PlotEnv, HeatmapLayer} from 'src/minard'
import {useLayer} from 'src/minard/utils/useLayer'
import {bin2d} from 'src/minard/utils/bin2d'
import HeatmapSquares from 'src/minard/components/HeatmapSquares'

interface Props {
  env: PlotEnv
  x: string
  y: string
  colors: string[]
  binSize?: number
}

export const Heatmap: SFC<Props> = ({env, x, y, colors, binSize = 20}) => {
  const {
    width,
    height,
    innerWidth,
    innerHeight,
    xDomain,
    yDomain,
    baseLayer: {
      table: baseTable,
      scales: {x: xScale, y: yScale},
    },
  } = env

  const layer = useLayer(
    env,
    (): Partial<HeatmapLayer> => {
      const [table, mappings] = bin2d(
        baseTable,
        x,
        xDomain,
        y,
        yDomain,
        width,
        height,
        binSize
      )

      return {type: 'heatmap', table, mappings, colors}
    },
    [baseTable, x, xDomain, y, yDomain, width, height, binSize, colors]
  )

  if (!layer) {
    return null
  }

  return (
    <HeatmapSquares
      layer={layer}
      width={innerWidth}
      height={innerHeight}
      xScale={xScale}
      yScale={yScale}
    />
  )
}
