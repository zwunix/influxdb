// Libraries
import React, {PureComponent, ChangeEvent} from 'react'
import {connect} from 'react-redux'

// Types
import {MeState} from 'src/types/v2'
import {
  Button,
  ComponentSize,
  ComponentStatus,
  Columns,
} from '@influxdata/clockface'
import {Form, Input, Panel, Grid} from 'src/clockface'
import Test from 'src/me/components/account/Test'
import TestThing from 'src/me/components/account/TestThing'

interface StateProps {
  me: MeState
}

interface State {
  me: MeState
  visible: boolean
}

export class Settings extends PureComponent<StateProps, State> {
  constructor(props) {
    super(props)
    this.state = {
      me: this.props.me,
      visible: false,
    }
  }

  public render() {
    const {me, visible} = this.state

    return (
      <Grid>
        <Grid.Row>
          <Grid.Column widthXS={Columns.Six}>
            <Panel>
              <Panel.Header title="About Me">
                <Button text="Edit About Me" />
              </Panel.Header>
              <Panel.Body>
                <Form>
                  <Form.Element label="Username">
                    <Input
                      value={me.name}
                      testID="nameInput"
                      titleText="Username"
                      size={ComponentSize.Small}
                      status={ComponentStatus.Disabled}
                      onChange={this.handleChangeInput}
                    />
                  </Form.Element>
                </Form>
                <Test
                  timeout={250}
                  visible={visible}
                  trigger={() => (
                    <button onClick={this.handleShow}>Toggle</button>
                  )}
                >
                  {state => (
                    <TestThing
                      state={state}
                      color="#00ffcc"
                      onDismiss={this.handleDismiss}
                    />
                  )}
                </Test>
              </Panel.Body>
            </Panel>
          </Grid.Column>
        </Grid.Row>
      </Grid>
    )
  }

  private handleShow = () => {
    this.setState({visible: true})
  }

  private handleDismiss = () => {
    this.setState({visible: false})
  }

  private handleChangeInput = (_: ChangeEvent<HTMLInputElement>): void => {
    //  console.log('changing: ', e)
  }
}

const mstp = ({me}) => ({
  me,
})

export default connect<StateProps>(mstp)(Settings)
