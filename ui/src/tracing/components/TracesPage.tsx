import React, {Component} from 'react'
import {Page} from 'src/pageLayout'
import {IndexList, EmptyState} from 'src/clockface'

const mockTraces = [
  {
    baggage: {},
    fields: {
      duration: '-2562047h47m16.854775808s',
    },
    finish_time: '2019-01-31T13:34:26.020694-08:00',
    operation: 'testtest',
    span_id: '035574bfb932c000',
    start_time: '2019-01-31T13:34:26.02068-08:00',
    tags: {
      hello: 'hellotag',
    },
    trace_id: '035574bfb932c001',
  },
  {
    baggage: {},
    fields: {
      duration: '-2562047h47m16.854775808s',
    },
    finish_time: '2019-01-31T13:34:26.020694-08:00',
    operation: 'testtest',
    span_id: '035574bfb932c000',
    start_time: '2019-01-31T13:34:26.02068-08:00',
    tags: {
      hello: 'hellotag',
    },
    trace_id: 'sdfsdfsfdsfsf',
  },
]

class TracesPage extends Component {
  public render() {
    return (
      <Page titleTag="Traces">
        <Page.Header fullWidth={false}>
          <Page.Header.Left>
            <Page.Title title="Traces" />
          </Page.Header.Left>
          <Page.Header.Right />
        </Page.Header>
        <Page.Contents fullWidth={false} scrollable={true}>
          <div className="col-xs-12">{this.list}</div>
        </Page.Contents>
      </Page>
    )
  }

  private get list() {
    return (
      <IndexList>
        <IndexList.Header>
          <IndexList.HeaderCell columnName="Trace ID" />
          <IndexList.HeaderCell columnName="Start" width="20%" />
          <IndexList.HeaderCell columnName="Stop" width="20%" />
          <IndexList.HeaderCell columnName="Operation" width="20%" />
        </IndexList.Header>
        <IndexList.Body columnCount={3} emptyState={this.emptyState}>
          {this.traces}
        </IndexList.Body>
      </IndexList>
    )
  }

  private get traces() {
    return mockTraces.map(trace => (
      <IndexList.Row key={trace.trace_id}>
        <IndexList.Cell>
          <a href="#">{trace.trace_id}</a>
        </IndexList.Cell>
        <IndexList.Cell>{trace.start_time}</IndexList.Cell>
        <IndexList.Cell>{trace.finish_time}</IndexList.Cell>
        <IndexList.Cell>{trace.operation}</IndexList.Cell>
      </IndexList.Row>
    ))
  }

  private get emptyState() {
    return (
      <EmptyState>
        <EmptyState.Text text="Sorry Mario your traces are in another castle!" />
      </EmptyState>
    )
  }
}

export default TracesPage
