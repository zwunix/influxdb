import {useEffect, useRef, EffectCallback, InputIdentityList} from 'react'

export const useMountedEffect = (
  effect: EffectCallback,
  inputs?: InputIdentityList
) => {
  const isFirstRender = useRef(true)

  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false

      return
    }

    return effect()
  }, inputs)
}
