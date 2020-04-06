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
	"google.golang.org/genproto/googleapis/rpc/code"
)

var p4rtClients = make(map[p4rtClientKey]P4RuntimeClient)

type P4RuntimeClient interface {
	SetMastership(electionId p4.Uint128) error
	GetForwardingPipelineConfig() (*p4.ForwardingPipelineConfig, error)
	SetForwardingPipelineConfig(p4InfoPath, deviceConfigPath string) error
	Write(update *p4.Update)
	SetWriteTraceChan(traceChan chan WriteTrace)
}

type p4rtClientKey struct {
	host string
	deviceId uint64
}

type p4rtClient struct {
	client p4.P4RuntimeClient
	stream p4.P4Runtime_StreamChannelClient
	deviceId uint64
	electionId p4.Uint128
	writes chan *p4.Update
	writeTraceChan chan WriteTrace
}

func (c *p4rtClient) Init() (err error) {
	// Initialize stream for mastership and packet I/O
	c.stream, err = c.client.StreamChannel(context.Background())
	if err != nil {
		return
	}
	go func() {
		for {
			res, err := c.stream.Recv()
			if err != nil {
				fmt.Printf("stream recv error: %v\n", err)
			} else if arb := res.GetArbitration(); arb != nil {
				if code.Code(arb.Status.Code) == code.Code_OK {
					fmt.Println("client is master")
				} else {
					fmt.Println("client is not master")
				}
			} else {
				fmt.Printf("stream recv: %v\n", res)
			}

		}
	}()

	// Initialize Write thread
	c.writes = make(chan *p4.Update, WRITE_BUFFER_SIZE)
	for i := 0; i < NUM_PARALLEL_WRITERS; i++ {
		go c.ListenForWrites()
	}

	return
}

func GetP4RuntimeClient(host string, deviceId uint64) (P4RuntimeClient, error) {
	key := p4rtClientKey{
		host:     host,
		deviceId: deviceId,
	}

	// First, return a P4RT client if one exists
	if p4rtClient, ok := p4rtClients[key]; ok {
		return p4rtClient, nil
	}

	// Second, check to see if we can reuse the gRPC connection for a new P4RT client
	conn, err := GetConnection(host)
	if err != nil {
		return nil, err
	}
	client :=  &p4rtClient{
		client: p4.NewP4RuntimeClient(conn),
		deviceId: deviceId,
	}
	err = client.Init()
	if err != nil {
		return nil, err
	}
	p4rtClients[key] = client
	return client, nil
}
