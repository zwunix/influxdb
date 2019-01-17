// Libraries
import {get} from 'lodash'

// APIs
import {executeQuery, ExecuteFluxQueryResult} from 'src/shared/apis/v2/query'
import {parseResponse} from 'src/shared/parsing/flux/response'

// Utils
import {formatTagFilterCall} from 'src/shared/utils/queryBuilder'

// Types
import {InfluxLanguage, BuilderConfig} from 'src/types/v2'
import {WrappedCancelablePromise} from 'src/types/promises'

export const SEARCH_DURATION = '30d'
export const LIMIT = 200

type CancelableQuery = WrappedCancelablePromise<string[]>

function findBuckets(url: string): CancelableQuery {
  const query = `buckets()
  |> sort(columns: ["name"])
  |> limit(n: ${LIMIT})`

  const response = executeQuery(url, query, InfluxLanguage.Flux)
  return parseQuery(response, resp => extractCol(resp, 'name'))
}

function findKeys(
  url: string,
  bucket: string,
  tagsSelections: BuilderConfig['tags'],
  searchTerm: string = ''
): CancelableQuery {
  const tagFilters = formatTagFilterCall(tagsSelections)
  const searchFilter = formatSearchFilterCall(searchTerm)
  const previousKeyFilter = formatTagKeyFilterCall(tagsSelections)

  const query = `from(bucket: "${bucket}")
  |> range(start: -${SEARCH_DURATION})${tagFilters}
  |> keys()
  |> group()
  |> distinct()
  |> keep(columns: ["_value"])${searchFilter}${previousKeyFilter}
  |> sort()
  |> limit(n: ${LIMIT})`

  const response = executeQuery(url, query, InfluxLanguage.Flux)

  return parseQuery(response, resp => extractCol(resp, '_value'))
}

function findValues(
  url: string,
  bucket: string,
  tagsSelections: BuilderConfig['tags'],
  key: string,
  searchTerm: string = ''
): CancelableQuery {
  const tagFilters = formatTagFilterCall(tagsSelections)
  const searchFilter = formatSearchFilterCall(searchTerm)

  const query = `from(bucket: "${bucket}")
    |> range(start: -${SEARCH_DURATION})${tagFilters}
    |> group(columns: ["${key}"])
    |> distinct(column: "${key}")
    |> group()
    |> keep(columns: ["_value"])${searchFilter}
    |> sort()
    |> limit(n: ${LIMIT})`

  const response = executeQuery(url, query, InfluxLanguage.Flux)

  return parseQuery(response, resp => extractCol(resp, '_value'))
}

function extractCol(resp: ExecuteFluxQueryResult, colName: string): string[] {
  const tables = parseResponse(resp.csv)
  const data = get(tables, '0.data', [])

  if (!data.length) {
    return []
  }

  const colIndex = data[0].findIndex(d => d === colName)

  if (colIndex === -1) {
    throw new Error(`could not find column "${colName}" in response`)
  }

  const colValues = []

  for (let i = 1; i < data.length; i++) {
    colValues.push(data[i][colIndex])
  }

  return colValues
}

function formatTagKeyFilterCall(tagsSelections: BuilderConfig['tags']) {
  const keys = tagsSelections.map(({key}) => key)

  if (!keys.length) {
    return ''
  }

  const fnBody = keys.map(key => `r._value != "${key}"`).join(' and ')

  return `\n  |> filter(fn: (r) => ${fnBody})`
}

function formatSearchFilterCall(searchTerm: string) {
  if (!searchTerm) {
    return ''
  }

  return `\n  |> filter(fn: (r) => r._value =~ /(?i:${searchTerm})/)`
}

function parseQuery(
  query: WrappedCancelablePromise<ExecuteFluxQueryResult>,
  parser: (result: ExecuteFluxQueryResult) => string[]
): CancelableQuery {
  const {promise: resp, cancel} = query
  const parse = async result => parser(await result)

  return {promise: parse(resp), cancel}
}

export class QueryBuilderFetcher {
  private findBucketsQuery: CancelableQuery
  private findKeysQueries: CancelableQuery[] = []
  private findValuesQueries: CancelableQuery[] = []

  public async findBuckets(url: string): Promise<string[]> {
    if (this.findBucketsQuery) {
      this.findBucketsQuery.cancel()
    }
    this.findBucketsQuery = findBuckets(url)

    return this.findBucketsQuery.promise
  }

  public async findKeys(
    index: number,
    url: string,
    bucket: string,
    tagsSelections: BuilderConfig['tags'],
    searchTerm: string = ''
  ): Promise<string[]> {
    this.cancelFindKeys(index)
    this.findKeysQueries[index] = findKeys(
      url,
      bucket,
      tagsSelections,
      searchTerm
    )
    return this.findKeysQueries[index].promise
  }

  public cancelFindKeys(index) {
    if (this.findKeysQueries[index]) {
      this.findKeysQueries[index].cancel()
    }
  }

  public async findValues(
    index: number,
    url: string,
    bucket: string,
    tagsSelections: BuilderConfig['tags'],
    key: string,
    searchTerm: string = ''
  ): Promise<string[]> {
    this.cancelFindValues(index)
    this.findValuesQueries[index] = findValues(
      url,
      bucket,
      tagsSelections,
      key,
      searchTerm
    )
    return this.findValuesQueries[index].promise
  }

  public cancelFindValues(index) {
    if (this.findValuesQueries[index]) {
      this.findValuesQueries[index].cancel()
    }
  }
}
