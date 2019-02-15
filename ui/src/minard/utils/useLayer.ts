import uuid from 'uuid'
import {useEffect, useRef, InputIdentityList} from 'react'
import {PlotEnv, Layer} from 'src/minard'

import {registerLayer, unregisterLayer} from 'src/minard/utils/plotEnvActions'

export const useLayer = (
  env: PlotEnv,
  layerFactory: () => Layer,
  inputs?: InputIdentityList
) => {
  const {current: layerKey} = useRef(uuid.v4())

  useEffect(() => {
    env.dispatch(registerLayer(layerKey, layerFactory()))

    return () => env.dispatch(unregisterLayer(layerKey))
  }, inputs)

  return env.layers[layerKey]
}
