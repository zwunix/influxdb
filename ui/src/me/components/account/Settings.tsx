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
import {Form, Input, Panel, Grid, Dropdown} from 'src/clockface'

interface StateProps {
  me: MeState
}

interface State {
  me: MeState
  dropdownSelected: string
}

const items = [
  'a',
  'b',
  'c',
  'd',
  'e',
  'f',
  'g',
  'h',
  'i',
  'j',
  'k',
  'l',
  'm',
  'n',
  'o',
  'p',
]

export class Settings extends PureComponent<StateProps, State> {
  constructor(props) {
    super(props)
    this.state = {
      me: this.props.me,
      dropdownSelected: items[0],
    }
  }

  public render() {
    const {me} = this.state

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
                {this.dropdown}
              </Panel.Body>
            </Panel>
          </Grid.Column>
        </Grid.Row>
      </Grid>
    )
  }

  private handleChangeInput = (_: ChangeEvent<HTMLInputElement>): void => {
    //  console.log('changing: ', e)
  }

  private get dropdown(): JSX.Element {
    const {dropdownSelected} = this.state

    return (
      <Dropdown
        selectedID={dropdownSelected}
        onChange={this.handleDropdownChange}
      >
        {items.map(item => (
          <Dropdown.Item key={item} value={item} id={item}>
            {item}
          </Dropdown.Item>
        ))}
      </Dropdown>
    )
  }

  private handleDropdownChange = (dropdownSelected: string): void => {
    this.setState({dropdownSelected})
  }
}

const mstp = ({me}) => ({
  me,
})

export default connect<StateProps>(mstp)(Settings)
