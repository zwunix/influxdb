import React, {PureComponent} from 'react'

export default class ExportOverlayMaybe extends PureComponent {
  public render() {
    return (
      <>
        {this.props.children}
        {/*overlay technnology goes here */}
      </>
    )
  }
}
