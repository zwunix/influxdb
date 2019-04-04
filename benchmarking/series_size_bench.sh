STEPS=$1
shift
TOKEN=$1
shift
query=$*

make > /dev/null && echo "make successful"
NPOINTS=1
for i in {0..6}
do
let NPOINTS=NPOINTS*10
./bin/darwin/influxd generate simple --t 10 --p $NPOINTS --clean all --bucket my-bucket --org my-org

./bin/darwin/influxd >/dev/null &
INFLUX_PID=$!
sleep 5

#$HOME/.local/bin/bench  -v 2 "curl http://localhost:9999/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d '$query'" --output example.html --csv "bench_$NPOINTS.csv"

hyperfine -r 10 --shell bash -u millisecond --export-markdown series_size.md "curl http://localhost:9999/api/v2/query?org=my-org -XPOST -sS -H 'Authorization: Token $TOKEN' -H 'accept:application/csv' -H 'content-type:application/vnd.flux' -d '$query'" 


kill $INFLUX_PID
done
