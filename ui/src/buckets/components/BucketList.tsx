// Libraries
import React, {PureComponent} from 'react'
import {withRouter, WithRouterProps} from 'react-router'
import {connect} from 'react-redux'
import {get} from 'lodash'

// Components
import UpdateBucketOverlay from 'src/buckets/components/UpdateBucketOverlay'
import BucketRow, {PrettyBucket} from 'src/buckets/components/BucketRow'
import {Overlay, IndexList, Sort} from 'src/clockface'
import {SortTypes} from 'src/buckets/components/BucketsTab'

// Actions
import {setBucketInfo} from 'src/dataLoaders/actions/steps'

// Selectors
import {getSortedResource} from 'src/shared/selectors/sort'

// Types
import {OverlayState} from 'src/types'
import {DataLoaderType} from 'src/types/dataLoaders'
import {setDataLoadersType} from 'src/dataLoaders/actions/dataLoaders'
import {AppState} from 'src/types'

type SortKey = keyof PrettyBucket

interface OwnProps {
  buckets: PrettyBucket[]
  emptyState: JSX.Element
  onUpdateBucket: (b: PrettyBucket) => void
  onDeleteBucket: (b: PrettyBucket) => void
  onFilterChange: (searchTerm: string) => void
  sortKey: string
  sortDirection: Sort
  sortType: SortTypes
  onClickColumn: (mextSort: Sort, sortKey: SortKey) => void
}

interface DispatchProps {
  onSetBucketInfo: typeof setBucketInfo
  onSetDataLoadersType: typeof setDataLoadersType
}

interface StateProps {
  dataLoaderType: DataLoaderType
  sortedIDs: string[]
}

type Props = OwnProps & StateProps & DispatchProps

interface State {
  bucketID: string
  bucketOverlayState: OverlayState
  sortedIDs: string[]
}

class BucketList extends PureComponent<Props & WithRouterProps, State> {
  constructor(props) {
    super(props)
    const bucketID = get(this, 'props.buckets.0.id', null)

    this.state = {
      bucketID,
      bucketOverlayState: OverlayState.Closed,
      sortedIDs: this.props.sortedIDs,
    }
  }

  componentDidUpdate(prevProps) {
    const {buckets, sortedIDs, sortKey, sortDirection} = this.props

    if (
      prevProps.sortDirection !== sortDirection ||
      prevProps.sortKey !== sortKey ||
      prevProps.buckets.length !== buckets.length
    ) {
      this.setState({sortedIDs})
    }
  }

  public render() {
    const {emptyState, sortKey, sortDirection, onClickColumn} = this.props

    return (
      <>
        <IndexList>
          <IndexList.Header>
            <IndexList.HeaderCell
              sortKey={this.headerKeys[0]}
              sort={sortKey === this.headerKeys[0] ? sortDirection : Sort.None}
              columnName="Name"
              width="40%"
              onClick={onClickColumn}
            />
            <IndexList.HeaderCell
              sortKey={this.headerKeys[1]}
              sort={sortKey === this.headerKeys[1] ? sortDirection : Sort.None}
              columnName="Retention"
              width="40%"
              onClick={onClickColumn}
            />
            <IndexList.HeaderCell columnName="" width="20%" />
          </IndexList.Header>
          <IndexList.Body columnCount={3} emptyState={emptyState}>
            {this.listBuckets}
          </IndexList.Body>
        </IndexList>
        <Overlay visible={this.isBucketOverlayVisible}>
          <UpdateBucketOverlay
            bucket={this.bucket}
            onCloseModal={this.handleCloseModal}
            onUpdateBucket={this.handleUpdateBucket}
          />
        </Overlay>
      </>
    )
  }

  private get headerKeys(): SortKey[] {
    return ['name', 'ruleString']
  }

  private get listBuckets(): JSX.Element[] {
    const {buckets, onDeleteBucket, onFilterChange} = this.props
    const {sortedIDs} = this.state

    return sortedIDs.map(id => {
      const bucket = buckets.find(b => b.id === id)
      return (
        bucket && (
          <BucketRow
            key={bucket.id}
            bucket={bucket}
            onEditBucket={this.handleStartEdit}
            onDeleteBucket={onDeleteBucket}
            onAddData={this.handleStartAddData}
            onUpdateBucket={this.handleUpdateBucket}
            onFilterChange={onFilterChange}
          />
        )
      )
    })
  }

  private get bucket(): PrettyBucket {
    return this.props.buckets.find(b => b.id === this.state.bucketID)
  }

  private handleCloseModal = () => {
    this.setState({bucketOverlayState: OverlayState.Closed})
  }

  private handleStartEdit = (bucket: PrettyBucket) => {
    this.setState({bucketID: bucket.id, bucketOverlayState: OverlayState.Open})
  }

  private handleStartAddData = (
    bucket: PrettyBucket,
    dataLoaderType: DataLoaderType,
    link: string
  ) => {
    const {onSetBucketInfo, onSetDataLoadersType, router} = this.props
    onSetBucketInfo(
      bucket.organization,
      bucket.organizationID,
      bucket.name,
      bucket.id
    )

    this.setState({
      bucketID: bucket.id,
    })

    onSetDataLoadersType(dataLoaderType)
    router.push(link)
  }

  private get isBucketOverlayVisible(): boolean {
    const {bucketID, bucketOverlayState} = this.state
    return !!bucketID && bucketOverlayState === OverlayState.Open
  }

  private handleUpdateBucket = async (updatedBucket: PrettyBucket) => {
    await this.props.onUpdateBucket(updatedBucket)
    this.setState({bucketOverlayState: OverlayState.Closed})
  }
}

const mstp = (state: AppState, props: OwnProps): StateProps => {
  return {
    dataLoaderType: state.dataLoading.dataLoaders.type,
    sortedIDs: getSortedResource(props.buckets, props),
  }
}

const mdtp: DispatchProps = {
  onSetBucketInfo: setBucketInfo,
  onSetDataLoadersType: setDataLoadersType,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(withRouter<Props>(BucketList))
