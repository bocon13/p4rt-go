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
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	p4 "github.com/p4lang/p4runtime/proto/p4/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

var MAX_BATCH_SIZE = 200
var NUM_PARALLEL_WRITERS = 1
var WRITE_BUFFER_SIZE = MAX_BATCH_SIZE * NUM_PARALLEL_WRITERS * 10

type p4Write struct {
	update   *p4.Update
	response chan *p4.Error
}

type WriteTrace struct {
	BatchSize int
	Duration  time.Duration
	Errors    []*p4.Error
}

func (c *p4rtClient) Write(update *p4.Update) <-chan *p4.Error {
	res := make(chan *p4.Error, 1)
	c.writes <- p4Write{
		update:   proto.Clone(update).(*p4.Update),
		response: res,
	}
	return res
}

func (c *p4rtClient) SetWriteTraceChan(traceChan chan WriteTrace) {
	c.writeTraceChan = traceChan
}

func (c *p4rtClient) ListenForWrites() {
	for {
		writes := make([]p4Write, MAX_BATCH_SIZE)
		var currBatchSize int
		writes[0] = <-c.writes // wait for the first write in the batch
	batch: // read as much as we can from the write channel into the batch
		for currBatchSize = 1; currBatchSize < MAX_BATCH_SIZE; currBatchSize++ {
			select {
			case write := <-c.writes:
				writes[currBatchSize] = write
			default: // no write update is immediately available
				break batch
			}
		}

		// Build the batch write request
		updates := make([]*p4.Update, currBatchSize)
		for i := range updates {
			updates[i] = writes[i].update
		}
		req := &p4.WriteRequest{
			DeviceId:   c.deviceId,
			ElectionId: &c.electionId,
			Updates:    updates,
		}
		// Write the request
		start := time.Now()
		_, err := c.client.Write(context.Background(), req)
		// ignore the write response; it is an empty message (details, if any, are in err)
		go processWriteResponse(writes, err, currBatchSize, start, c.writeTraceChan)
	}
}

func processWriteResponse(writes []p4Write, err error, batchSize int, start time.Time, traceChan chan WriteTrace) {
	duration := time.Since(start)
	errors := ParseP4RuntimeWriteError(err, batchSize)
	// Send p4.Errors to waiting channels
	for i := range errors {
		writes[i].response <- errors[i]
	}

	if traceChan != nil {
		trace := WriteTrace{
			BatchSize: batchSize,
			Duration:  duration,
			Errors:    errors,
		}
		select {
		case traceChan <- trace: // put trace into the channel unless it is full
		default:
			fmt.Println("Write trace channel full. Discarding trace")
		}
	}

}

func ParseP4RuntimeWriteError(err error, batchSize int) []*p4.Error {
	errors := make([]*p4.Error, batchSize)
	var code int32
	var message = ""
	if err != nil {
		grpcError := status.Convert(err).Proto() // TODO consider status.FromError()
		if grpcError.GetCode() == int32(codes.Unknown) && batchSize > 0 && len(grpcError.GetDetails()) == batchSize {
			// gRPC error may contain p4.Errors
			for i := range grpcError.Details {
				p4Err := p4.Error{}
				unmarshallErr := ptypes.UnmarshalAny(grpcError.Details[i], &p4Err)
				if unmarshallErr != nil {
					// Unmarshalling p4.Error failed (construct a synthetic p4.Error)
					p4Err = p4.Error{
						CanonicalCode: int32(codes.Internal),
						Message:       unmarshallErr.Error(),
						Space:         "p4rt-go",
					}
				}
				errors[i] = &p4Err
			}
			return errors
		}
		message = grpcError.GetMessage()
	} else {
		code = int32(codes.OK)
	}

	// If the error does not have p4.Errors, build a stand-in p4.Error for all requests
	p4Error := &p4.Error{
		CanonicalCode: code,
		Message:       message,
	}
	for i := range errors {
		errors[i] = p4Error
	}
	return errors
}

func (c *p4rtClient) RemainingWrites() bool {
	return len(c.writes) > 0
}
