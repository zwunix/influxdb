STEPS=$1
shift
TOKEN=5CJjFZBVRme8YRCEsSbonzgmz-Y50HsggB8QVPrUI9_zno_Zz9YzbZynZ94BWI6Dkz2CC5p63agcOgLRQtQLlw==
shift
query=$*

NPOINTS=1000000
KEYSCNT="10"

NVALUES=1
for i in {2..6}
do
    measurement=$i"t10v1000000p"
    echo $measurement
    hyperfine -r 10 --shell bash -u millisecond --export-markdown series_size.md "curl http://localhost:8086/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d 'from(bucket:\"my-bucket\") |> range(start:-30d)|> filter(fn: (r) => r._measurement == $measurement)'" 


#kill $INFLUX_PID
done
