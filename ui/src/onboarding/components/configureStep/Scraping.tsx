// Libraries
import React, {PureComponent} from 'react'

// Components
import {Form, Input, Dropdown} from 'src/clockface'

interface Props {}

class Scraping extends PureComponent<Props> {
  public render() {
    return (
      <Form>
        <Form.Element label="Interval">
          <Input widthPixels={200} />
        </Form.Element>
        <Form.Element label="Bucket">
          {/* <Dropdown
            selectedID={'1'}
            onChange={this.handleOnClick}
            widthPixels={200}
          >
            <Dropdown.Item key={1} value={1} id={'1'}>
              defbuck
            </Dropdown.Item>
            ))}
          </Dropdown> */}
        </Form.Element>
      </Form>
    )
  }
}
// private handleOnClick = (value: any) => {
//   console.log(value)
// }

export default Scraping
