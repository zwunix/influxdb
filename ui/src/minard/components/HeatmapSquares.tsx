import React, {useRef, useLayoutEffect, SFC} from 'react'

import {Scale, HeatmapLayer} from 'src/minard'
import {clearCanvas} from 'src/minard/utils/clearCanvas'

interface Props {
  layer: HeatmapLayer
  width: number
  height: number
  xScale: Scale<number, number>
  yScale: Scale<number, number>
}

const drawSquares = (
  canvas: HTMLCanvasElement,
  {layer, width, height, xScale, yScale}: Props
) => {
  clearCanvas(canvas, width, height)

  const {
    table,
    mappings,
    scales: {fill: fillScale},
  } = layer

  const context = canvas.getContext('2d')

  for (let i = 0; i < table.length; i++) {
    const xMin = table.columns[mappings.xMin].data[i]
    const xMax = table.columns[mappings.xMax].data[i]
    const yMin = table.columns[mappings.yMin].data[i]
    const yMax = table.columns[mappings.yMax].data[i]
    const fill = table.columns[mappings.fill].data[i]

    const squareX = xScale(xMin)
    const squareY = yScale(yMax)
    const squareWidth = xScale(xMax) - squareX
    const squareHeight = yScale(yMin) - squareY

    context.beginPath()
    context.rect(squareX, squareY, squareWidth, squareHeight)
    context.globalAlpha = fill === 0 ? 0 : 1
    context.fillStyle = fillScale(fill)
    context.fill()
  }
}

const HeatmapSquares: SFC<Props> = props => {
  const canvas = useRef<HTMLCanvasElement>(null)

  useLayoutEffect(() => drawSquares(canvas.current, props))

  return <canvas className="minard-layer heatmap" ref={canvas} />
}

export default React.memo(HeatmapSquares)
