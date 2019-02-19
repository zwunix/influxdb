import React, {PureComponent} from 'react'
import moment from 'moment'

interface Props {
  time: Date
}

export default class extends PureComponent<Props> {
  public render() {
    return <>{this.relativeTime}</>
  }

  private get relativeTime(): string {
    return moment(this.props.time).from(moment())
  }
}
