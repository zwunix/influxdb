// Libraries
import React, {useMemo, SFC} from 'react'
import {AutoSizer} from 'react-virtualized'
import {Plot as MinardPlot, Heatmap as MinardHeatmap} from 'src/minard'

// Utils
import {toMinardTable} from 'src/shared/utils/toMinardTable'

// Types
import {HeatmapView} from 'src/types/v2/dashboards'
import {FluxTable} from 'src/types'

interface Props {
  tables: FluxTable[]
  properties: HeatmapView
}

const Heatmap: SFC<Props> = ({tables, properties}) => {
  const {table} = useMemo(() => toMinardTable(tables), [tables])
  const {xColumn, yColumn, binSize, colors} = properties

  return (
    <AutoSizer>
      {({width, height}) => {
        if (!width || !height) {
          return null
        }

        return (
          <MinardPlot table={table} width={width} height={height}>
            {env => (
              <MinardHeatmap
                env={env}
                x={xColumn}
                y={yColumn}
                binSize={binSize}
                colors={colors}
              />
            )}
          </MinardPlot>
        )
      }}
    </AutoSizer>
  )
}

export default Heatmap
