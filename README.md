# P4Runtime Go Client

Client library and flow write performance tester

## Setup (one time)
*Install Go* (>= 1.13.3)
https://golang.org/doc/install

*Install Protoc* (>= 3.2.0)
You only need the C++ version
https://developers.google.com/protocol-buffers/

`export GOPATH=~/go` (if you changed it)

```
bash <(curl -s https://raw.githubusercontent.com/bocon13/p4rt-go/master/setup.sh)
```

## Building for BMv2

Build the test binary
```
cd $HOME/go/src/github.com/bocon13/p4rt-go/
go build -tags bmv2 -o p4rt_test_bmv2 bin/main.go
```

Run Stratum BMv2:
```
docker run --privileged --rm -it -p 50001:50001 opennetworking/mn-stratum
```

Then, you can run the test:
```
./p4rt_test_bmv2 \
 -target localhost:50001 \
 -p4info test/bmv2/p4info.txt \
 -deviceConfig test/bmv2/bmv2.json \
 -count 10000 \
 -verbose
```

## Building for Tofino

Build the test binary
```
GOOS=linux go build -tags tofino -o p4rt_test_tofino bin/main.go
```

Start Stratum on your Tofino switch

Then, you can run the test:
```
./p4rt_test_tofino \
 -target localhost:28000 \
 -p4info test/montara/p4info.txt \
 -deviceConfig test/montara/tofino.bin,test/montara/context.json \
 -count 1000 \
 -verbose
```

Notes:
- If you use the test files, they were compiled against SDE 9.0.0
- Remember to update the target string to match the IP of your switch (or run the test on the box)
- Update GOOS to match the operating system of where you will run the test binary
- You can use any P4 program/compiler version that you want, just be sure to update the paths