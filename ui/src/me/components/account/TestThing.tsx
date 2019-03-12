// Libraries
import React, {Component} from 'react'
import classnames from 'classnames'

// Styles
import './TestThing.scss'

interface Props {
  state: string
  color: string
  onDismiss: () => void
}

export default class TestThing extends Component<Props> {
  render() {
    const {state, color, onDismiss} = this.props
    const className = classnames('test-thing', {
      'test-thing--entering': state === 'entering',
      'test-thing--entered': state === 'entered',
      'test-thing--exiting': state === 'exiting',
    })

    const style = {backgroundColor: color}

    return (
      <div className={className} style={style}>
        <p>{state}</p>
        <div className="test-thing--close" onClick={onDismiss}>
          ×
        </div>
      </div>
    )
  }
}
