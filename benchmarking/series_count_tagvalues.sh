STEPS=$1
shift
TOKEN=5CJjFZBVRme8YRCEsSbonzgmz-Y50HsggB8QVPrUI9_zno_Zz9YzbZynZ94BWI6Dkz2CC5p63agcOgLRQtQLlw==
shift
query=$*

NPOINTS=1000000
KEYSCNT="10"

NVALUES=1
for i in {0..4}
do
    measurement="\"1t$((10**i))v1000000p\""
 #   echo $measurement
 hyperfine -r 10 --shell bash -u millisecond --export-csv 1TagNValues.csv "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == $measurement)'" "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == $measurement)|> group(columns:[\"tag0\"])'"   "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == $measurement) |> window(every:5m) |> count()'"

 #this is just prohibitively slow
# "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == $measurement)|> map(fn: (r) => r._value * 1000.0) '"
 
#hyperfine -P K 2 6 -r 10 --shell bash -u millisecond --export-csv nTags10Values.csv "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'import \"influxdata/influxdb/v1\" v1.measurementTagKeys(bucket:\"db\", measurement: \"m0\")'"

hyperfine -r 10 --shell bash -u millisecond --export-csv 1TagNValuesInfluxQL.csv "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT * FROM $measurement'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT * FROM $measurement group by \"tag0\"'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT (\"v0\" * 1000) FROM $measurement'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT count(*) FROM $measurement group by time(5m)'"  "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SHOW TAG KEYS FROM $measurement'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SHOW TAG VALUES FROM $measurement with key =~ /tag.*/'"

# kill $INFLUX_PID
done
