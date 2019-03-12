// Libraries
import React, {Component} from 'react'
import {Transition} from 'react-transition-group'

interface Props {
  children: (state: string) => JSX.Element
  timeout: number
  visible: boolean
  trigger: () => JSX.Element
}

export default class Test extends Component<Props> {
  render() {
    const {children, timeout, visible, trigger} = this.props

    return (
      <>
        {trigger()}
        <Transition
          in={visible}
          timeout={timeout}
          unmountOnExit={true}
          mountOnEnter={true}
        >
          {children}
        </Transition>
      </>
    )
  }
}
