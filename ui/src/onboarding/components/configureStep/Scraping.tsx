// Libraries
import React, {PureComponent, ChangeEvent} from 'react'
import {connect} from 'react-redux'

// Components
import {
  Form,
  Input,
  Columns,
  Grid,
  ComponentSize,
  MultipleInput,
  Dropdown,
  InputType,
} from 'src/clockface'
import FancyScrollbar from 'src/shared/components/fancy_scrollbar/FancyScrollbar'

// Actions
import {
  setScrapingInterval,
  addScrapingURL,
  removeScrapingURL,
  updateScrapingURL,
  setScrapingBucket,
} from 'src/onboarding/actions/dataLoaders'
import {AppState} from 'src/types/v2/index'
import {SetupParams} from 'src/onboarding/apis'

interface OwnProps {}

interface DispatchProps {
  setScrapingInterval: typeof setScrapingInterval
  setScrapingBucket: typeof setScrapingBucket
  addScrapingURL: typeof addScrapingURL
  removeScrapingURL: typeof removeScrapingURL
  updateScrapingURL: typeof updateScrapingURL
}

interface StateProps {
  interval: string
  bucket: string
  urls: string[]
  setupParams: SetupParams
}

interface State {
  intervalInput: string
}

type Props = OwnProps & DispatchProps & StateProps

class Scraping extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props)

    this.state = {
      intervalInput: '',
    }
  }
  public render() {
    const {intervalInput} = this.state
    const {bucket} = this.props

    return (
      <Form onSubmit={this.handleOnClick}>
        <div className="wizard-step--scroll-area">
          <FancyScrollbar autoHide={false}>
            <div className="wizard-step--scroll-content">
              <h3 className="wizard-step--title">Add Scraper</h3>
              <h5 className="wizard-step--sub-title">
                Scrapers collect data from multiple targets at regular intervals
                and to write to a bucket
              </h5>
              <Grid>
                <Grid.Row>
                  <Grid.Column
                    widthXS={Columns.Six}
                    widthMD={Columns.Five}
                    offsetMD={Columns.One}
                  >
                    <Form.Element label="Interval">
                      <Input
                        type={InputType.Text}
                        value={intervalInput}
                        onChange={this.handleChange}
                        titleText="Interval"
                        size={ComponentSize.Medium}
                        autoFocus={true}
                      />
                    </Form.Element>
                  </Grid.Column>
                  <Grid.Column
                    widthXS={Columns.Six}
                    widthMD={Columns.Five}
                    offsetMD={Columns.One}
                  >
                    <Form.Element label="Bucket">
                      <Dropdown
                        selectedID={bucket}
                        onChange={this.handleBucket}
                      >
                        {this.dropdownBuckets}
                      </Dropdown>
                    </Form.Element>
                  </Grid.Column>
                  <Grid.Column
                    widthXS={Columns.Twelve}
                    widthMD={Columns.Ten}
                    offsetMD={Columns.One}
                  >
                    <MultipleInput
                      onAddRow={this.handleAddRow}
                      onDeleteRow={this.handleRemoveRow}
                      onEditRow={this.handleEditRow}
                      tags={this.tags}
                      title={'Add URL'}
                      helpText={''}
                    />
                  </Grid.Column>
                </Grid.Row>
              </Grid>
            </div>
          </FancyScrollbar>
        </div>
      </Form>
    )
  }

  private get dropdownBuckets(): JSX.Element[] {
    const {setupParams} = this.props

    const buckets = [setupParams.bucket]

    // This is a hacky fix
    return buckets.map(b => (
      <Dropdown.Item key={b} value={b} id={b}>
        {b}
      </Dropdown.Item>
    ))
  }

  private handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value
    this.setState({intervalInput: value})
    this.props.setScrapingInterval(value)
  }

  private handleAddRow = (item: string) => {
    this.props.addScrapingURL(item)
  }

  private handleRemoveRow = (item: string) => {
    this.props.removeScrapingURL(item)
  }

  private handleEditRow = (index: number, item: string) => {
    this.props.updateScrapingURL(index, item)
  }

  private get tags(): Array<{name: string; text: string}> {
    const {urls} = this.props
    return urls.map(v => {
      return {text: v, name: v}
    })
  }

  private handleBucket = (bucket: string) => {
    this.props.setScrapingBucket(bucket)
  }

  private handleOnClick = (value: any) => {
    console.log(value)
  }
}

const mstp = ({
  onboarding: {
    steps: {setupParams},
    dataLoaders: {
      scraper: {interval, bucket, urls},
    },
  },
}: AppState) => {
  return {setupParams, interval, bucket, urls}
}

const mdtp: DispatchProps = {
  setScrapingInterval,
  setScrapingBucket,
  addScrapingURL,
  removeScrapingURL,
  updateScrapingURL,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(Scraping)
