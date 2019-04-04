STEPS=$1
shift
TOKEN=$1
shift
query=$*

make > /dev/null && echo "make successful"
NPOINTS=1000000
NVALUES=1
for i in {0..6}
do
    let NVALUES=(10**i)
    let NPOINTS=1000000/NVALUES
./bin/darwin/influxd generate simple --t $NVALUES --p $NPOINTS --clean all --bucket my-bucket --org my-org

./bin/darwin/influxd >/dev/null &
INFLUX_PID=$!
sleep 5

hyperfine -r 10 --shell bash -u millisecond --export-markdown series_size.md "curl http://localhost:9999/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d '$query'" 


kill $INFLUX_PID
done
