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

const specTemplate = {
  $schema: 'https://vega.github.io/schema/vega-lite/v2.0.json',
  description: 'This chart was made with vega embed!',
  mark: 'point',
  encoding: {
    x: {
      field: 'time',
      type: 'temporal',
      axis: {title: 'chronological stuff'},
    },
    y: {
      field: 'value',
      type: 'quantitative',
      axis: {title: 'numerical stuff'},
    },
  },
}

const opt = {
  // width: 100,
  // height: 100,
  renderer: 'svg',
}

export default class VegaVis extends Component<Props> {
  public componentDidMount() {
    const {vegaObj} = this.props
    const spec = {...specTemplate, ...vegaObj, data: this.dataMaker()}
    vegaEmbed('#vis', spec, opt)
  }

  public render() {
    console.log('rendering VegaVis')
    console.log(this.props.tables)
    const {vegaObj} = this.props
    const spec = {...specTemplate, ...vegaObj, data: this.dataMaker()}
    console.log('spec', spec)
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

      const formatted = data.map(d => ({
        time: d[timeIndex],
        value: d[valIndex],
      }))

      const theData = {
        name: 'awesomeData',
        values: formatted,
      }
      return theData
    }
    return null
  }
}
