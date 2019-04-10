STEPS=$1
shift
TOKEN=5CJjFZBVRme8YRCEsSbonzgmz-Y50HsggB8QVPrUI9_zno_Zz9YzbZynZ94BWI6Dkz2CC5p63agcOgLRQtQLlw==
shift
query=$*

NPOINTS=1000000
KEYSCNT="10"

NVALUES=1
#for i in {2..6}
#do
#    measurement=$i"t10v1000000p"
 #   echo $measurement
    #hyperfine -P K 2 6 -r 10 --shell bash -u millisecond --export-csv nTags10Values.csv "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == \"{K}t10v1000000p\")'" 

#hyperfine -P K 2 6 -r 10 --shell bash -u millisecond --export-csv nTags10Values.csv "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == \"{K}t10v1000000p\")|> group(columns:[\"tag0\"])'" "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == \"{K}t10v1000000p\")|> map(fn: (r) => r._value * 1000.0) '"  "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"db\") |> range(start:-30d)|> filter(fn: (r) => r._m == \"{K}t10v1000000p\") |> window(every:5m) '"

#hyperfine -P K 2 6 -r 10 --shell bash -u millisecond --export-csv nTags10Values.csv "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'import \"influxdata/influxdb/v1\" v1.measurementTagKeys(bucket:\"db\", measurement: \"m0\")'"

#hyperfine -P K 2 6 -r 10 --shell bash -u millisecond --export-csv nTags10ValuesInfluxQL.csv "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT * FROM \"{K}t10v1000000p\"'" 

#hyperfine -P K 2 6 -r 10 --shell bash -u millisecond --export-csv nTags10ValuesInfluxQL.csv "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT (\"v0\" * 1000) FROM \"{K}t10v1000000p\"'"  "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT * FROM \"{K}t10v1000000p\" group by \"tag0\"'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT * FROM \"{K}t10v1000000p\" group by time(5m)'"

hyperfine -P K 2 6 -r 10 --shell bash -u millisecond --export-csv nTags10ValuesInfluxQL.csv "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SELECT count(*) FROM \"{K}t10v1000000p\" group by time(5m)'"  "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SHOW TAG KEYS FROM \"{K}t10v1000000p\"'" "curl -G 'http://localhost:8086/query?db=db' --data-urlencode 'q=SHOW TAG VALUES FROM \"{K}t10v1000000p\" with key =~ /tag.*/'"

# kill $INFLUX_PID
#done
