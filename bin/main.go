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

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/bocon13/p4rt-go/p4rt"
	"github.com/golang/protobuf/proto"
	p4 "github.com/p4lang/p4runtime/proto/p4/v1"
	"google.golang.org/grpc/codes"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var writeReples sync.WaitGroup
var failedWrites uint32

func main() {

	target := flag.String("target", "localhost:28000", "")
	verbose := flag.Bool("verbose", false, "")
	p4info := flag.String("p4info", "", "")
	count := flag.Uint64("count", 1, "")
	deviceConfig := flag.String("deviceConfig", "", "")

	flag.Parse()

	client, err := p4rt.GetP4RuntimeClient(*target, 1)
	if err != nil {
		panic(err)
	}

	err = client.SetMastership(p4.Uint128{High: 0, Low: 1})
	if err != nil {
		panic(err)
	}

	err = client.SetForwardingPipelineConfig(*p4info, *deviceConfig)
	if err != nil {
		panic(err)
	}

	//config, err := client.GetForwardingPipelineConfig()
	//if err != nil {
	//	panic(err)
	//}

	// Set up write tracing for test
	writeTraceChan := make(chan p4rt.WriteTrace, 100)
	client.SetWriteTraceChan(writeTraceChan)
	doneChan := make(chan bool)
	go func() {
		var writeCount, lastCount uint64
		printInterval := 1 * time.Second
		ticker := time.Tick(printInterval)
		for {
			select {
			case trace := <-writeTraceChan:
				writeCount += uint64(trace.BatchSize)
				if writeCount >= *count {
					doneChan <- true
					return
				}
			case <-ticker:
				if *verbose {
					fmt.Printf("\033[2K\rWrote %d of %d (~%.1f flows/sec)...",
						writeCount, *count, float64(writeCount-lastCount)/printInterval.Seconds())
					lastCount = writeCount
				}
			}
		}
	}()

	// Send the flow entries
	writeReples.Add(int(*count))
	start := time.Now()
	SendTableEntries(client, *count)

	// Wait for all writes to finish
	<-doneChan
	duration := time.Since(start).Seconds()
	fmt.Printf("\033[2K\r%f seconds, %d writes, %f writes/sec\n",
		duration, *count, float64(*count)/duration)
	writeReples.Wait()
	fmt.Printf("Number of failed writes: %d\n", failedWrites)
}

func SendTableEntries(p4rt p4rt.P4RuntimeClient, count uint64) {
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
		Type: p4.Update_INSERT,
		Entity: &p4.Entity{Entity: &p4.Entity_TableEntry{
			TableEntry: &p4.TableEntry{
				TableId: 33574274, // FabricIngress.forwarding.mpls
				Match:   match,
				Action: &p4.TableAction{Type: &p4.TableAction_Action{Action: &p4.Action{
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

	for i := uint64(0); i < count; i++ {
		//update.GetEntity().GetTableEntry().GetMatch()[0].FieldId = uint32(i % 2)
		matchField := update.GetEntity().GetTableEntry().GetMatch()[0].GetExact()
		matchField.Value = Uint64(i)[5:8] // mpls_label is 20 bits
		res := p4rt.Write(update)
		go CountFailed(proto.Clone(update).(*p4.Update), res)
	}
}

func CountFailed(update *p4.Update, res <-chan *p4.Error) {
	err := <-res
	if err.CanonicalCode != int32(codes.OK) { // write failed
		atomic.AddUint32(&failedWrites, 1)
		fmt.Fprintf(os.Stderr, "%v -> %v\n", update, err.GetMessage())
	}
	writeReples.Done()
}

func Uint64(v uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, v)
	return bytes
}
