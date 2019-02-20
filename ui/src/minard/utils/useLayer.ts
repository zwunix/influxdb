import uuid from 'uuid'
import {useEffect, useRef, DependencyList} from 'react'
import {PlotEnv, Layer} from 'src/minard'

import {registerLayer, unregisterLayer} from 'src/minard/utils/plotEnvActions'

/*
  Register a layer in the plot environment. A layer can optionally specify its
  own data, color scheme, and data-to-aesthetic mappings.
*/
export const useLayer = <L extends Layer>(
  env: PlotEnv,
  layerFactory: () => Partial<L>,
  inputs?: DependencyList
): L => {
  const {current: layerKey} = useRef(uuid.v4())

  useEffect(() => {
    env.dispatch(registerLayer(layerKey, layerFactory() as L))

    return () => env.dispatch(unregisterLayer(layerKey))
  }, inputs)

  return env.layers[layerKey] as L
}
