import {extent} from 'd3-array'
import {useRef} from 'react'

/*
  Use the supplied `domain`, or fallback to computing a domain from the
  supplied `col`. Preserves the logical identity of returned domain.
*/
export const useDomain = (
  domain: [number, number],
  col: number[]
): [number, number] => {
  const lastCol = useRef(null)
  const lastDomain = useRef(domain)

  if (
    domain &&
    lastDomain.current &&
    domain[0] === lastDomain.current[0] &&
    domain[1] === lastDomain.current[1]
  ) {
    return lastDomain.current
  }

  if (domain) {
    lastDomain.current = domain

    return domain
  }

  if (lastCol.current === col) {
    return lastDomain.current
  }

  lastCol.current = col
  lastDomain.current = extent(col)

  return lastDomain.current
}
