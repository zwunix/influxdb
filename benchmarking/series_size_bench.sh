TOKEN=5CJjFZBVRme8YRCEsSbonzgmz-Y50HsggB8QVPrUI9_zno_Zz9YzbZynZ94BWI6Dkz2CC5p63agcOgLRQtQLlw==

for i in {2..7}
do
    measurement="\"1t10v$((10**i))p\""

 hyperfine -r 10 --shell bash -u millisecond --export-csv 1TagNValues.csv "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == $measurement)'" "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == $measurement)|> group(columns:[\"tag0\"])'" "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == $measurement)|> map(fn: (r) => r._value * 1000.0) '"  "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == $measurement) |> window(every:5m) |> count()'"
    
    hyperfine -r 10 --shell bash -u millisecond --export-csv 1TagNPointsInfluxQL.csv "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT * FROM $measurement'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT * FROM $measurement group by \"tag0\"'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT (\"v0\" * 1000) FROM $measurement'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT count(*) FROM $measurement group by time(5m)'"  "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SHOW TAG KEYS FROM $measurement'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SHOW TAG VALUES FROM $measurement with key =~ /tag.*/'"

done
