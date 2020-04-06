
## Setup (one time)


## Building for BMv2

`go build -tags bmv2 -o p4rt_test_bmv2 bin/main.go`

Then, you can run with:

```
./p4rt_test \
 -target localhost:50001 \
 -p4info /tmp/test/p4info.txt \
 -deviceConfig /tmp/test/bmv2.json \
 -count 10000 \
 -verbose
```

## Building for Tofino


./p4rt_test \
-p4info fabric-tofino/tmp/fabric/montara_sde_9_0_0/p4info.txt \
-deviceConfig fabric-tofino/tmp/fabric/montara_sde_9_0_0/pipe/tofino.bin,fabric-tofino/tmp/fabric/montara_sde_9_0_0/pipe/context.json \
-count 100 \
-verbose \
2>/tmp/out.txt