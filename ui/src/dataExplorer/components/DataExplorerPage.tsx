// Libraries
import React, {SFC} from 'react'

// Components
import DataExplorer from 'src/dataExplorer/components/DataExplorer'
import {Page} from '@influxdata/clockface'
import SaveAsButton from 'src/dataExplorer/components/SaveAsButton'
import VisOptionsButton from 'src/timeMachine/components/VisOptionsButton'
import ViewTypeDropdown from 'src/timeMachine/components/view_options/ViewTypeDropdown'
import PageTitleWithOrg from 'src/shared/components/PageTitleWithOrg'

// Constants
import {PAGE_TITLE_SUFFIX} from 'src/shared/constants'

const DataExplorerPage: SFC = ({children}) => {
  return (
    <Page loadingTitleTag={`Data Explorer${PAGE_TITLE_SUFFIX}`}>
      {children}
      <Page.Header fullWidth={true}>
        <Page.Header.Left>
          <PageTitleWithOrg title="Data Explorer" />
        </Page.Header.Left>
        <Page.Header.Right>
          <ViewTypeDropdown />
          <VisOptionsButton />
          <SaveAsButton />
        </Page.Header.Right>
      </Page.Header>
      <Page.Contents fullWidth={true} scrollable={false}>
        <DataExplorer />
      </Page.Contents>
    </Page>
  )
}

export default DataExplorerPage
