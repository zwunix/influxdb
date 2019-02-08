// Libraries
import React, {PureComponent} from 'react'
import {Page} from 'src/pageLayout'
import RenamablePageTitle from 'src/pageLayout/components/RenamablePageTitle'

interface Props {
  orgName: string
  onUpdateOrg: (name: string) => void
}

export class OrgHeader extends PureComponent<Props> {
  public render() {
    const {orgName, onUpdateOrg} = this.props

    return (
      <Page.Header fullWidth={false}>
        <Page.Header.Left>
          <RenamablePageTitle
            name={orgName}
            maxLength={70}
            placeholder="Name this Organization"
            onRename={onUpdateOrg}
          />
        </Page.Header.Left>
        <Page.Header.Right />
      </Page.Header>
    )
  }
}
