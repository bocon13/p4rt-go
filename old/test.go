/*
 * Copyright 2020-present Brian O'Connor
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// This was used for Brian's original performance test
// File is included for reference

package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	p4_config_v1 "github.com/p4lang/p4runtime/proto/p4/config/v1"
	p4 "github.com/p4lang/p4runtime/proto/p4/v1"
	"google.golang.org/grpc"
	"io/ioutil"
	"os"
	"sync"
	"time"
)


func main() {

	target := flag.String("target", "localhost:28000", "")
	batchSize := flag.Uint64("batchSize", 1, "")
	iterations := flag.Uint64("iterations", 1, "")
	parallel := flag.Bool("parallel", false, "")
	verbose := flag.Bool("verbose", false, "")
	p4info := flag.String("p4info", "", "")
	bmv2Bin := flag.String("bmv2bin", "", "")
	tofinoBin := flag.String("tofinobin", "", "")
	tofinoContext := flag.String("tofinoctx", "", "")

	flag.Parse()
	fmt.Println(*target, *batchSize, *iterations)

	// Connect to the switch
	conn, err := grpc.Dial(*target, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	p4RuntimeClient := p4.NewP4RuntimeClient(conn)

	// Open the stream channel and send mastership request
	stream, err := p4RuntimeClient.StreamChannel(context.Background())
	if err != nil {
		panic(err)
	}
	mastershipReq := &p4.StreamMessageRequest{
		Update: &p4.StreamMessageRequest_Arbitration{
			Arbitration: &p4.MasterArbitrationUpdate{
				DeviceId: 1,
				ElectionId: &p4.Uint128{
					High: 1,
					Low: 1,
				},
			},
		},
	}
	err = stream.Send(mastershipReq)

	// Send the pipeconf
	SendPipeconf(p4RuntimeClient, *p4info, *tofinoBin, *tofinoContext, *bmv2Bin)

	// Send the flow entries
	var wg sync.WaitGroup
	if *parallel {
		wg.Add(int(*iterations))
	}
	start := time.Now()
	SendTableEntries(p4RuntimeClient, *batchSize, *iterations, *parallel, &wg, *verbose)
	if *parallel {
		wg.Wait()
	}
	fmt.Println(time.Since(start).Seconds())
}

func SendPipeconf(p4rt p4.P4RuntimeClient, p4infoPath string, tofinoBinPath string, tofinoContextPath string, bmv2BinPath string) {
	p4infoBytes, err := ioutil.ReadFile(p4infoPath)
	if err != nil {
		panic(err)
	}
	p4info := &p4_config_v1.P4Info{}
	if err = proto.UnmarshalText(string(p4infoBytes), p4info); err != nil {
		panic(err)
	}

	var p4Bin []byte
	if tofinoBinPath != "" {
		// Build the target binary
		pipeconfName := "name"

		tofinoBin, err := os.Open(tofinoBinPath)
		if err != nil {
			panic(err)
		}
		defer tofinoBin.Close()
		tofinoBinInfo, err := tofinoBin.Stat()
		if err != nil {
			panic(err)
		}

		tofinoContext, err := os.Open(tofinoContextPath)
		if err != nil {
			panic(err)
		}
		defer tofinoContext.Close()
		tofinoContextInfo, err := tofinoContext.Stat()
		if err != nil {
			panic(err)
		}

		binLen := len(pipeconfName) + int(tofinoBinInfo.Size()) + int(tofinoContextInfo.Size()) + 12 // 3 * 32bit int
		p4Bin = make([]byte, binLen)

		i := 0
		binary.LittleEndian.PutUint32(p4Bin[i:], uint32(len(pipeconfName)))
		i += 4
		if b := copy(p4Bin[i:], pipeconfName); b != len(pipeconfName) {
			panic("pipeconf name copy failed")
		}
		i += len(pipeconfName)

		binary.LittleEndian.PutUint32(p4Bin[i:], uint32(tofinoBinInfo.Size()))
		i += 4
		if b, err := tofinoBin.Read(p4Bin[i:]); err != nil {
			panic(err)
		} else if b != int(tofinoBinInfo.Size()) {
			panic("tofino bin copy failed")
		}
		i += int(tofinoBinInfo.Size())

		binary.LittleEndian.PutUint32(p4Bin[i:], uint32(tofinoContextInfo.Size()))
		i += 4
		if b, err := tofinoContext.Read(p4Bin[i:]); err != nil {
			panic(err)
		} else if b != int(tofinoContextInfo.Size()) {
			panic("tofino context copy failed")
		}
		i += int(tofinoContextInfo.Size())

		if i != binLen {
			panic("failed to build p4 device config for tofino")
		}
	} else if bmv2BinPath != "" {
		bmv2Bin, err := os.Open(bmv2BinPath)
		if err != nil {
			panic(err)
		}
		defer bmv2Bin.Close()
		bmv2BinInfo, err := bmv2Bin.Stat()
		if err != nil {
			panic(err)
		}

		p4Bin = make([]byte, int(bmv2BinInfo.Size()))
		if b, err := bmv2Bin.Read(p4Bin); err != nil {
			panic(err)
		} else if b != int(bmv2BinInfo.Size()) {
			panic("bmv2 bin copy failed")
		}
	}

	pipeconf := &p4.SetForwardingPipelineConfigRequest{
		DeviceId:   1,
		RoleId:     0, // not used
		ElectionId: &p4.Uint128{
			High: 1,
			Low: 1,
		},
		Action:     p4.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT,
		Config:     &p4.ForwardingPipelineConfig{
			P4Info: p4info,
			P4DeviceConfig: p4Bin,
		},
	}
	if res, err := p4rt.SetForwardingPipelineConfig(context.Background(), pipeconf); err != nil {
		fmt.Println(res.String())
		panic(err)
	}
}

func SendTableEntries(p4rt p4.P4RuntimeClient, batchSize uint64, iters uint64, parallel bool, wg *sync.WaitGroup, verbose bool) {
	req := &p4.WriteRequest{
		DeviceId: 1,
		ElectionId: &p4.Uint128{
			High: 1,
			Low: 1,
		},
	}

	match := []*p4.FieldMatch{
		{
			FieldId:        1, // mpls_label
			FieldMatchType: &p4.FieldMatch_Exact_{&p4.FieldMatch_Exact{}},
		},
		// more fields...
		//{
		//	FieldId:        0,
		//	FieldMatchType: &p4.FieldMatch_Exact_{
		//		Exact: &p4.FieldMatch_Exact{[]byte{4, 5, 6, 7}}},
		//},
	}

	update := &p4.Update{
		Type:   p4.Update_INSERT,
		Entity: &p4.Entity{&p4.Entity_TableEntry{
			TableEntry: &p4.TableEntry{
				TableId: 33574274, // FabricIngress.forwarding.mpls
				Match: match,
				Action: &p4.TableAction{&p4.TableAction_Action{Action: &p4.Action{
					ActionId: 16827758, // pop_mpls_and_next
					Params: []*p4.Action_Param{
						{
							ParamId: 1,              // next_id
							Value:   Uint64(0)[0:4], // 32 bits
						},
					},
				}}},
			},
		}},
	}

	req.Updates = make([]*p4.Update, batchSize)
	matchFields := make([]*p4.FieldMatch_Exact, batchSize)
	for i := uint64(0); i < batchSize; i++ {
		req.Updates[i] = proto.Clone(update).(*p4.Update)
		matchFields[i] = req.Updates[i].GetEntity().GetTableEntry().GetMatch()[0].GetExact()
	}

	for i := uint64(0); i < iters; i++ {
		for j := uint64(0); j < batchSize; j++ {
			x := i * batchSize + j
			matchFields[j].Value = Uint64(x)[5:8] // mpls_label is 20 bits
		}
		run := func(req *p4.WriteRequest) {
			if parallel {
				defer wg.Done()
			}
			start := time.Now()
			if res, err := p4rt.Write(context.Background(), req); err != nil {
				fmt.Println(res.String())
				panic(err)
			}
			if verbose {
				//fmt.Println(time.Since(start))
				fmt.Fprint(os.Stderr, time.Since(start).Seconds(), "\n")
			}
		}
		if parallel {
			go run(proto.Clone(req).(*p4.WriteRequest))
		} else {
			run(req)
		}

	}

}

func Uint64(v uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, v)
	return bytes
}