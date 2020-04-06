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

package p4rt

import (
	"context"
	"fmt"
	p4 "github.com/p4lang/p4runtime/proto/p4/v1"
	"time"
)

var MAX_BATCH_SIZE = 200
var NUM_PARALLEL_WRITERS = 1
var WRITE_BUFFER_SIZE = MAX_BATCH_SIZE * NUM_PARALLEL_WRITERS * 10

func (c *p4rtClient) Write(update *p4.Update) {
	//TODO add a callback mechanism to let the original writer know if success/failure
	//FIXME to make sure that the called doesn't change the update under us:
	//   proto.Clone(update).(*p4.Update)
	c.writes <- update
}

type WriteTrace struct {
	BatchSize int
	Duration time.Duration
}

func (c *p4rtClient) SetWriteTraceChan(traceChan chan WriteTrace) {
	c.writeTraceChan = traceChan
}

func (c *p4rtClient) ListenForWrites() {
	for {
		updates := make([]*p4.Update, MAX_BATCH_SIZE)
		var i int
		batch: // read as much as we can from the write chan and send
		for i = 0; i < MAX_BATCH_SIZE; i++ {
			select {
			case update := <-c.writes:
				updates[i] = update
			default:
				break batch
			}
		}
		if i == 0 {
			continue // batch is empty
		}
		req := &p4.WriteRequest{
			DeviceId:   c.deviceId,
			ElectionId: &c.electionId,
			Updates:    updates[0:i],
		}
		start := time.Now()
		_, err := c.client.Write(context.Background(), req)
		// ignore the response; it is an empty message
		if err != nil {
			fmt.Printf("error writing to device: %v\n", err)
			//TODO add a callback mechanism to let the original writer know if success/failure
		} else if c.writeTraceChan != nil {
			trace := WriteTrace{
				BatchSize: i,
				Duration:  time.Since(start),
			}
			select {
			case c.writeTraceChan <- trace: // put trace into the channel unless it is full
			default:
				fmt.Println("Write trace channel full. Discarding trace")
			}
		}
	}
}


func (c *p4rtClient) RemainingWrites() bool {
	return len(c.writes) > 0
}