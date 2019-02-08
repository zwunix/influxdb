import React, {Component} from 'react'
import vegaEmbed from 'vega-embed'
import {FluxTable} from 'src/types'
import _ from 'lodash'

interface Props {
  tables: FluxTable[]
  vegaObj: object
}

interface DataValues {
  a: string
  b: number
}

interface Dataz {
  name: string
  values: DataValues[]
}

const specTemplate = {}
//   $schema: 'https://vega.github.io/schema/vega-lite/v3.0.json',
//   description: 'This chart was made with vega embed!',
//   mark: {
//     type: 'line',
//     point: {
//       filled: true,
//       fill: 'red',
//     },
//   },
//   encoding: {
//     x: {
//       field: 'time',
//       type: 'temporal',
//       axis: {
//         title: 'chronological stuff',
//         titleColor: '#fff',
//         titleFontSize: 14,
//       },
//     },
//     y: {
//       field: 'value',
//       type: 'quantitative',
//       axis: {title: 'numerical stuff', titleColor: '#fff', titleFontSize: 14},
//     },
//   },
// }

const opt = {
  width: 500,
  height: 500,
  renderer: 'svg',
}

export default class VegaVis extends Component<Props> {
  public componentDidMount() {
    const {vegaObj} = this.props
    const spec = {...specTemplate, ...vegaObj, data: this.dataMaker()}
    vegaEmbed('#vis', spec, opt)
  }

  public render() {
    // console.log('rendering VegaVis')
    // console.log(this.props.tables)
    const {vegaObj} = this.props
    const spec = {...specTemplate, ...vegaObj, data: this.dataMaker()}
    // console.log('spec', spec)
    vegaEmbed('#vis', spec, opt).then(res =>
      res.view.insert('awesomeData', this.dataMaker()).run()
    )
    return <div id="vis" />
  }

  private dataMaker = (): Dataz | null => {
    const {tables} = this.props
    if (tables.length > 0) {
      const data = _.get(tables, '0.data')
      const header = data.shift()
      const timeIndex = _.indexOf(header, '_time')
      const valIndex = _.indexOf(header, '_value')

      const values = data.map(d => ({
        time: d[timeIndex],
        value: d[valIndex],
        thing: 1,
      }))

      const theData = {
        name: 'awesomeData',
        values,
      }
      return theData
    }
    const theData = {
      name: 'awesomeData',
      values: [],
    }
    return theData
  }
}
