// Libraries
import React, {SFC} from 'react'
interface Props {
  children: JSX.Element
  visible: boolean
}

const OverlayTechnology: SFC<Props> = ({visible, children}) => {
  if (!visible) {
    return null
  }

  return (
    <div className="overlay-tech">
      <div className="overlay--dialog" data-test="overlay-children">
        {children}
      </div>
      <div className="overlay--mask" />
    </div>
  )
}

export default OverlayTechnology
